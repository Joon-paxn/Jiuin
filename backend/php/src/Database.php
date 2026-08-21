<?php
declare(strict_types=1);

namespace Jiuin;

use PDO;
use PDOException;
use RuntimeException;

final class Database
{
    public static function open(Config $config): PDO
    {
        $pdo = new PDO('sqlite:' . $config->databasePath, null, null, [
            PDO::ATTR_ERRMODE => PDO::ERRMODE_EXCEPTION,
            PDO::ATTR_DEFAULT_FETCH_MODE => PDO::FETCH_ASSOC,
            PDO::ATTR_EMULATE_PREPARES => false,
        ]);
        $pdo->exec('PRAGMA journal_mode=WAL; PRAGMA busy_timeout=5000; PRAGMA foreign_keys=ON;');
        $schema = file_get_contents(dirname(__DIR__, 2) . '/internal/core/schema.sql');
        if ($schema === false) {
            throw new RuntimeException('Shared database schema is unavailable');
        }
        $pdo->exec($schema);
        return $pdo;
    }

    /** The callback must only contain short database operations, never FFmpeg. */
    public static function immediate(PDO $pdo, callable $callback): mixed
    {
        $last = null;
        for ($attempt = 0; $attempt < 8; $attempt++) {
            try {
                $pdo->exec('BEGIN IMMEDIATE');
                try {
                    $result = $callback();
                    $pdo->exec('COMMIT');
                    return $result;
                } catch (\Throwable $exception) {
                    try { $pdo->exec('ROLLBACK'); } catch (PDOException) {}
                    throw $exception;
                }
            } catch (PDOException $exception) {
                $last = $exception;
                if (!str_contains(strtolower($exception->getMessage()), 'database is locked') && !str_contains(strtolower($exception->getMessage()), 'database is busy')) {
                    throw $exception;
                }
                usleep(($attempt + 1) * 40000);
            }
        }
        throw new RuntimeException('SQLite remained busy after retries', 0, $last);
    }
}
