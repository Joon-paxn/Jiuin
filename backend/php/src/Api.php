<?php
declare(strict_types=1);

namespace Jiuin;

use PDO;
use RuntimeException;

final class Api
{
    public function __construct(private readonly Config $config, private readonly PDO $pdo, private readonly MusicStore $music) {}

    public function handle(): void
    {
        $requestId = $_SERVER['HTTP_X_REQUEST_ID'] ?? ('req-' . bin2hex(random_bytes(8)));
        header('X-Request-ID: ' . substr($requestId, 0, 128));
        $method = $_SERVER['REQUEST_METHOD'] ?? 'GET';
        // HEAD shares the GET route contract. Nginx/PHP-FPM suppresses the
        // response body while retaining the availability and media headers.
        if ($method === 'HEAD') {
            $method = 'GET';
        }
        $path = parse_url($_SERVER['REQUEST_URI'] ?? '/', PHP_URL_PATH);
        if (!is_string($path)) { $this->error(400, 'invalid request path'); return; }
        try {
            match (true) {
                $method === 'GET' && ($path === '/health' || $path === '/api/v1/health') => $this->health(),
                $method === 'GET' && ($path === '/ready' || $path === '/api/v1/ready') => $this->ready(),
                $method === 'GET' && $path === '/api/v1/site' => $this->ok(['site' => $this->site(), 'copyright' => $this->copyright()]),
                $method === 'GET' && $path === '/api/v1/site/info' => $this->ok($this->site()),
                $method === 'GET' && $path === '/api/v1/site/copyright' => $this->ok($this->copyright()),
                $method === 'GET' && $path === '/api/v1/status' => $this->ok(['site' => 'online', 'api' => 'online', 'services' => [], 'checkedAt' => gmdate('c')]),
                $method === 'GET' && $path === '/api/v1/links' => $this->ok($this->config->externalLinks),
                $method === 'GET' && $path === '/api/v1/resources' => $this->ok($this->config->resources),
                $method === 'GET' && $path === '/api/v1/statistics' => $this->statistics(),
                $method === 'POST' && $path === '/api/v1/statistics/visit' => $this->recordVisit(),
                $method === 'GET' && $path === '/api/v1/background/random' => $this->randomBackground($requestId),
                $method === 'GET' && $path === '/api/v1/music' => $this->ok($this->music->listPublic()),
                $method === 'POST' && ($path === '/api/v1/admin/music/upload' || $path === '/api/v1/music/upload') => $this->upload(),
                $method === 'GET' && preg_match('#^/api/v1/music/([^/]+)$#', $path, $matches) === 1 => $this->getMusic($matches[1]),
                $method === 'GET' && preg_match('#^/media/music/([^/]+)/(cover|full|lite)$#', $path, $matches) === 1 => $this->media($matches[1], $matches[2]),
                default => $this->error(404, 'not found'),
            };
        } catch (RuntimeException $exception) {
            error_log('[jiuin-php] ' . $exception->getMessage());
            $this->error(503, 'service is unavailable');
        } catch (\Throwable $exception) {
            error_log('[jiuin-php] unexpected error: ' . $exception->getMessage());
            $this->error(500, 'internal server error');
        }
    }

    private function health(): void { $this->ok(['status' => 'ok']); }

    private function ready(): void
    {
        $this->pdo->query('SELECT 1')->fetchColumn();
        foreach (['tmp', 'original', 'full', 'lite', 'covers'] as $child) {
            if (!is_dir($this->config->storageDir . '/' . $child)) {
                $this->error(503, 'storage is unavailable');
                return;
            }
        }
        if (!$this->commandAvailable($this->config->ffmpegPath) || !$this->commandAvailable($this->config->ffprobePath)) {
            $this->error(503, 'media tooling is unavailable');
            return;
        }
        $this->ok(['status' => 'ready', 'dependencies' => ['database' => 'ok', 'storage' => 'ok', 'ffmpeg' => 'ok', 'ffprobe' => 'ok']]);
    }

    private function statistics(): void
    {
        $rows = $this->pdo->query('SELECT path,views,last_visited_at FROM page_statistics ORDER BY path')->fetchAll();
        $total = 0;
        $pages = array_map(function (array $row) use (&$total): array {
            $views = (int) $row['views']; $total += $views;
            return ['path' => $row['path'], 'views' => $views, 'lastVisitedAt' => $row['last_visited_at']];
        }, $rows);
        $this->ok(['totalViews' => $total, 'pages' => $pages]);
    }

    private function recordVisit(): void
    {
        if (!$this->authorized($this->config->serviceToken)) { $this->error(401, 'unauthorized'); return; }
        $data = json_decode((string) file_get_contents('php://input'), true);
        $path = is_array($data) && is_string($data['path'] ?? null) ? $data['path'] : '';
        if (!str_starts_with($path, '/') || strlen($path) > 512) { $this->error(400, 'a valid path is required'); return; }
        Database::immediate($this->pdo, function () use ($path): void {
            $this->pdo->prepare('INSERT INTO page_statistics(path,views,last_visited_at) VALUES(?,1,?) ON CONFLICT(path) DO UPDATE SET views=views+1,last_visited_at=excluded.last_visited_at')->execute([$path, gmdate('c')]);
        });
        $this->ok(['path' => $path]);
    }

    private function randomBackground(string $requestId): void
    {
        if ($this->config->backgroundUrls === []) { $this->error(404, 'no background is configured'); return; }
        $index = array_sum(array_map('ord', str_split($requestId))) % count($this->config->backgroundUrls);
        $url = $this->config->backgroundUrls[$index];
        if (!is_string($url)) { $this->error(503, 'background configuration is invalid'); return; }
        $this->ok(['url' => $url]);
    }

    private function getMusic(string $id): void
    {
        $music = $this->music->getPublic($id);
        $music === null ? $this->error(404, 'music was not found') : $this->ok($music);
    }

    private function upload(): void
    {
        if (!$this->authorized($this->config->adminToken)) { $this->error(401, 'unauthorized'); return; }
        $key = is_string($_SERVER['HTTP_IDEMPOTENCY_KEY'] ?? null) ? $_SERVER['HTTP_IDEMPOTENCY_KEY'] : '';
        if ($key === '') { $this->error(400, 'Idempotency-Key header is required'); return; }
        if ((int) ($_SERVER['CONTENT_LENGTH'] ?? 0) > 100 * 1024 * 1024) { $this->error(413, 'upload exceeds the allowed size'); return; }
        $file = $_FILES['file'] ?? null;
        if (!is_array($file) || ($file['error'] ?? UPLOAD_ERR_NO_FILE) !== UPLOAD_ERR_OK) { $this->error(400, 'file is required'); return; }
        try {
            $result = $this->music->createUpload($key, $_POST, $file);
        } catch (RuntimeException $exception) {
            if (str_contains($exception->getMessage(), 'different content')) {
                $this->error(409, $exception->getMessage());
                return;
            }
            if (str_contains($exception->getMessage(), 'required') || str_contains($exception->getMessage(), 'safe music')) {
                $this->error(400, $exception->getMessage());
                return;
            }
            throw $exception;
        }
        $this->respond($result['idempotentReplay'] ? 200 : 202, $result['idempotentReplay'] ? 'accepted' : 'accepted', $result);
    }

    private function media(string $id, string $quality): void
    {
        $stream = $this->music->openMedia($id, $quality);
        if ($stream === false) { $this->error(404, 'media was not found'); return; }
        $metadata = stream_get_meta_data($stream);
        $file = $metadata['uri'] ?? '';
        $size = filesize($file);
        if (!is_int($size)) { fclose($stream); $this->error(500, 'media is unavailable'); return; }
        $contentType = $quality === 'cover' ? 'image/jpeg' : 'audio/mpeg';
        header('Content-Type: ' . $contentType);
        header('Accept-Ranges: bytes');
        header('Cache-Control: public, max-age=3600');
        $range = $_SERVER['HTTP_RANGE'] ?? '';
        if (preg_match('/^bytes=(\d*)-(\d*)$/', $range, $matches) === 1) {
            $start = $matches[1] === '' ? 0 : (int) $matches[1];
            $end = $matches[2] === '' ? $size - 1 : min((int) $matches[2], $size - 1);
            if ($start > $end || $start >= $size) { fclose($stream); header('Content-Range: bytes */' . $size); http_response_code(416); return; }
            http_response_code(206); header("Content-Range: bytes {$start}-{$end}/{$size}"); header('Content-Length: ' . ($end - $start + 1)); fseek($stream, $start); $remaining = $end - $start + 1;
        } else { header('Content-Length: ' . $size); $remaining = $size; }
        while ($remaining > 0 && !feof($stream)) { $chunk = fread($stream, min(8192, $remaining)); if ($chunk === false) break; $remaining -= strlen($chunk); echo $chunk; }
        fclose($stream);
    }

    /** @return array{name:string,project:string,domain:string} */
    private function site(): array { return ['name' => $this->config->siteName, 'project' => $this->config->siteProject, 'domain' => $this->config->siteDomain]; }
    /** @return array{year:int,text:string} */
    private function copyright(): array { return ['year' => (int) gmdate('Y'), 'text' => $this->config->siteProject]; }
    private function commandAvailable(string $command): bool { exec(escapeshellarg($command) . ' -version 2>/dev/null', $output, $code); return $code === 0; }
    private function authorized(string $expected): bool { $provided = preg_replace('/^Bearer\\s+/i', '', $_SERVER['HTTP_AUTHORIZATION'] ?? ''); return $expected !== '' && is_string($provided) && hash_equals($expected, $provided); }
    private function ok(mixed $data): void { $this->respond(200, 'ok', $data); }
    private function error(int $code, string $message): void { $this->respond($code, $message); }
    private function respond(int $code, string $message, mixed $data = null): void { http_response_code($code); header('Content-Type: application/json; charset=utf-8'); header('Cache-Control: no-store'); echo json_encode($data === null ? ['code' => $code, 'message' => $message] : ['code' => $code, 'message' => $message, 'data' => $data], JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES); }
}
