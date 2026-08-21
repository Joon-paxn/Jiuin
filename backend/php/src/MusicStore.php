<?php
declare(strict_types=1);

namespace Jiuin;

use PDO;
use RuntimeException;

final class MusicStore
{
    public function __construct(private readonly Config $config, private readonly PDO $pdo) {}

    /** @return list<array<string,mixed>> */
    public function listPublic(): array
    {
        $statement = $this->pdo->query("SELECT id,title,artist,album,album_artist,genre,year,cover_path,duration_seconds,full_path,lite_path,full_size,lite_size,created_at FROM music WHERE state='ready' ORDER BY created_at DESC");
        return array_map(fn (array $row) => $this->publicMusic($row), $statement->fetchAll());
    }

    /** @return array<string,mixed>|null */
    public function getPublic(string $id): ?array
    {
        $statement = $this->pdo->prepare("SELECT id,title,artist,album,album_artist,genre,year,cover_path,duration_seconds,full_path,lite_path,full_size,lite_size,created_at FROM music WHERE id=? AND state='ready'");
        $statement->execute([$id]);
        $row = $statement->fetch();
        return is_array($row) ? $this->publicMusic($row) : null;
    }

    /** @return array{uploadId:string,taskId:string,musicId:string,state:string,idempotentReplay:bool} */
    public function createUpload(string $key, array $metadata, array $file): array
    {
        if ($key === '' || strlen($key) > 255) {
            throw new RuntimeException('Idempotency-Key header is required');
        }
        $title = $this->text($metadata['title'] ?? null);
        $artist = $this->text($metadata['artist'] ?? null);
        $name = is_string($file['name'] ?? null) ? $file['name'] : '';
        $tmp = is_string($file['tmp_name'] ?? null) ? $file['tmp_name'] : '';
        if ($title === '' || $artist === '' || basename($name) !== $name || $name === '' || !is_uploaded_file($tmp)) {
            throw new RuntimeException('title, artist, and a safe music file are required');
        }
        // The same deterministic IDs are used by Go. A retry after a crash
        // therefore targets the same storage objects and database rows.
        $musicId = $this->idForKey('music', $key);
        $uploadId = $this->idForKey('upload', $key);
        $taskId = $this->idForKey('task', $key);
        // Do not derive persistent paths from the client filename: a retry
        // with another extension must still use the same original object.
        $original = $this->config->storageDir . '/original/' . $musicId . '.upload';
        $hash = hash_file('sha256', $tmp);
        if (!is_string($hash)) {
            throw new RuntimeException('Cannot hash uploaded file');
        }
        $now = $this->now();
        $moved = false;
        try {
            $result = Database::immediate($this->pdo, function () use ($key, $tmp, $original, $musicId, $uploadId, $taskId, $hash, $now, $title, $artist, $metadata, $name, &$moved): array {
                $replay = $this->findUpload($key);
                if ($replay !== null) {
                    if (!hash_equals($replay['_contentSha256'], $hash)) {
                        throw new RuntimeException('Idempotency-Key was already used with different content');
                    }
                    unset($replay['_contentSha256']);
                    $replay['idempotentReplay'] = true;
                    return $replay;
                }
                if (!move_uploaded_file($tmp, $original)) {
                    throw new RuntimeException('Cannot persist uploaded file');
                }
                $moved = true;
                $this->pdo->prepare("INSERT INTO music (id,title,artist,album,album_artist,genre,year,source_name,source_path,original_path,state,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?, 'queued',?,?)")
                    ->execute([$musicId, $title, $artist, $this->text($metadata['album'] ?? null), $this->text($metadata['albumArtist'] ?? null), $this->text($metadata['genre'] ?? null), $this->text($metadata['year'] ?? null), $name, $name, $original, $now, $now]);
                $this->pdo->prepare('INSERT INTO upload_requests (idempotency_key,upload_id,task_id,music_id,content_sha256,created_at) VALUES (?,?,?,?,?,?)')
                    ->execute([$key, $uploadId, $taskId, $musicId, $hash, $now]);
                $this->pdo->prepare("INSERT INTO music_tasks (id,music_id,state,created_at,updated_at) VALUES (?,?,'queued',?,?)")
                    ->execute([$taskId, $musicId, $now, $now]);
                return ['uploadId' => $uploadId, 'taskId' => $taskId, 'musicId' => $musicId, 'state' => 'queued', 'idempotentReplay' => false];
            });
            return $result;
        } catch (\Throwable $exception) {
            if ($moved && is_file($original)) {
                @unlink($original);
            }
            throw $exception;
        }
    }

    /** @return resource|false */
    public function openMedia(string $id, string $quality)
    {
        $columns = ['cover' => 'cover_path', 'full' => 'full_path', 'lite' => 'lite_path'];
        if (!isset($columns[$quality])) {
            return false;
        }
        $statement = $this->pdo->prepare("SELECT {$columns[$quality]} AS path FROM music WHERE id=? AND state='ready'");
        $statement->execute([$id]);
        $path = $statement->fetchColumn();
        return is_string($path) && $path !== '' && is_file($path) ? fopen($path, 'rb') : false;
    }

    public function processOne(string $workerId): bool
    {
        $job = Database::immediate($this->pdo, function () use ($workerId): ?array {
            $now = $this->now();
            $statement = $this->pdo->prepare("SELECT t.id,t.music_id,m.original_path FROM music_tasks t JOIN music m ON m.id=t.music_id WHERE t.state='queued' OR (t.state='processing' AND t.lease_until < ?) ORDER BY t.created_at LIMIT 1");
            $statement->execute([$now]);
            $job = $statement->fetch();
            if (!is_array($job)) {
                return null;
            }
            $lease = gmdate('Y-m-d\\TH:i:s.u\\Z', time() + $this->config->processingLeaseSeconds);
            $claim = $this->pdo->prepare("UPDATE music_tasks SET state='processing',locked_by=?,lease_until=?,attempts=attempts+1,updated_at=? WHERE id=? AND (state='queued' OR (state='processing' AND lease_until < ?))");
            $claim->execute([$workerId, $lease, $now, $job['id'], $now]);
            return $claim->rowCount() === 1 ? $job : null;
        });
        if ($job === null) {
            return false;
        }
        $error = null;
        try {
            $this->transcode($job);
        } catch (\Throwable $exception) {
            $error = $exception;
        }
        $this->finishTask($job, $workerId, $error);
        if ($error !== null) {
            throw $error;
        }
        return true;
    }

    /** @param array{id:string,music_id:string,original_path:string} $job */
    private function transcode(array $job): void
    {
        $full = $this->config->storageDir . '/full/' . $job['music_id'] . '.mp3';
        $lite = $this->config->storageDir . '/lite/' . $job['music_id'] . '.mp3';
        $cover = $this->config->storageDir . '/covers/' . $job['music_id'] . '.jpg';
        $this->run([$this->config->ffmpegPath, '-y', '-i', $job['original_path'], '-vn', '-c:a', $this->config->outputCodec, '-b:a', $this->config->fullBitrate, $full]);
        $this->run([$this->config->ffmpegPath, '-y', '-i', $job['original_path'], '-vn', '-c:a', $this->config->outputCodec, '-b:a', $this->config->liteBitrate, $lite]);
        try {
            $this->run([$this->config->ffmpegPath, '-y', '-i', $job['original_path'], '-an', '-map', '0:v:0', '-frames:v', '1', $cover]);
        } catch (\Throwable) {
            $this->run([$this->config->ffmpegPath, '-y', '-f', 'lavfi', '-i', 'color=c=0x293241:s=1200x1200', '-frames:v', '1', $cover]);
        }
    }

    /** @param array{id:string,music_id:string} $job */
    private function finishTask(array $job, string $workerId, ?\Throwable $error): void
    {
        Database::immediate($this->pdo, function () use ($job, $workerId, $error): void {
            $now = $this->now();
            if ($error !== null) {
                $task = $this->pdo->prepare("UPDATE music_tasks SET state='failed',last_error=?,lease_until='',updated_at=? WHERE id=? AND locked_by=? AND state='processing'");
                $task->execute([substr($error->getMessage(), 0, 1000), $now, $job['id'], $workerId]);
                if ($task->rowCount() !== 1) {
                    throw new RuntimeException('Music task lease was lost');
                }
                $this->pdo->prepare("UPDATE music SET state='failed',updated_at=? WHERE id=?")->execute([$now, $job['music_id']]);
                return;
            }
            $full = $this->config->storageDir . '/full/' . $job['music_id'] . '.mp3';
            $lite = $this->config->storageDir . '/lite/' . $job['music_id'] . '.mp3';
            $cover = $this->config->storageDir . '/covers/' . $job['music_id'] . '.jpg';
            $duration = $this->probeDuration($job['original_path']);
            $task = $this->pdo->prepare("UPDATE music_tasks SET state='done',lease_until='',updated_at=? WHERE id=? AND locked_by=? AND state='processing'");
            $task->execute([$now, $job['id'], $workerId]);
            if ($task->rowCount() !== 1) {
                throw new RuntimeException('Music task lease was lost');
            }
            $this->pdo->prepare("UPDATE music SET state='ready',full_path=?,lite_path=?,cover_path=?,duration_seconds=?,full_size=?,lite_size=?,updated_at=? WHERE id=?")->execute([$full, $lite, $cover, $duration, filesize($full), filesize($lite), $now, $job['music_id']]);
        });
    }

    /** @return array{uploadId:string,taskId:string,musicId:string,state:string,idempotentReplay:bool,_contentSha256:string}|null */
    private function findUpload(string $key): ?array
    {
        $statement = $this->pdo->prepare('SELECT u.upload_id,u.task_id,u.music_id,t.state,u.content_sha256 FROM upload_requests u JOIN music_tasks t ON t.id=u.task_id WHERE u.idempotency_key=?');
        $statement->execute([$key]);
        $row = $statement->fetch();
        return is_array($row) ? ['uploadId' => $row['upload_id'], 'taskId' => $row['task_id'], 'musicId' => $row['music_id'], 'state' => $row['state'], 'idempotentReplay' => false, '_contentSha256' => $row['content_sha256']] : null;
    }

    /** @param array<string,mixed> $row @return array<string,mixed> */
    private function publicMusic(array $row): array
    {
        $data = ['id' => $row['id'], 'title' => $row['title'], 'artist' => $row['artist'], 'audio' => []];
        foreach (['album' => 'album', 'album_artist' => 'albumArtist', 'genre' => 'genre', 'year' => 'year', 'created_at' => 'createdAt'] as $column => $name) {
            if ($row[$column] !== '') $data[$name] = $row[$column];
        }
        if ($row['cover_path'] !== '') $data['cover'] = '/media/music/' . $row['id'] . '/cover';
        if ($row['full_path'] !== '') $data['audio']['full'] = '/media/music/' . $row['id'] . '/full';
        if ($row['lite_path'] !== '') $data['audio']['lite'] = '/media/music/' . $row['id'] . '/lite';
        if ($row['duration_seconds'] !== null) $data['durationSeconds'] = (float) $row['duration_seconds'];
        if ($row['full_size'] !== null) $data['fullSize'] = (int) $row['full_size'];
        if ($row['lite_size'] !== null) $data['liteSize'] = (int) $row['lite_size'];
        return $data;
    }

    /** @param list<string> $command */
    private function run(array $command): void
    {
        $escaped = implode(' ', array_map('escapeshellarg', $command)) . ' 2>&1';
        exec($escaped, $output, $code);
        if ($code !== 0) {
            throw new RuntimeException('FFmpeg failed: ' . substr(implode("\n", $output), 0, 1200));
        }
    }

    private function probeDuration(string $source): float
    {
        $command = implode(' ', array_map('escapeshellarg', [$this->config->ffprobePath, '-v', 'error', '-show_entries', 'format=duration', '-of', 'default=noprint_wrappers=1:nokey=1', $source])) . ' 2>&1';
        exec($command, $output, $code);
        $duration = isset($output[0]) ? (float) trim($output[0]) : -1;
        if ($code !== 0 || $duration < 0) {
            throw new RuntimeException('FFprobe failed to calculate duration');
        }
        return $duration;
    }

    private function idForKey(string $prefix, string $key): string { return $prefix . '_' . hash('sha256', $key); }
    private function now(): string { return gmdate('Y-m-d\\TH:i:s') . '.' . substr((string) microtime(), 2, 6) . 'Z'; }
    private function text(mixed $value): string { return is_string($value) ? trim($value) : ''; }
}
