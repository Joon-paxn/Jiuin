<?php
declare(strict_types=1);

use Jiuin\Api;
use Jiuin\Config;
use Jiuin\Database;
use Jiuin\MusicStore;

require dirname(__DIR__) . '/src/Config.php';
require dirname(__DIR__) . '/src/Database.php';
require dirname(__DIR__) . '/src/MusicStore.php';
require dirname(__DIR__) . '/src/Api.php';

try {
    $config = Config::fromEnvironment();
    $config->ensureStorage();
    $database = Database::open($config);
    $api = new Api($config, $database, new MusicStore($config, $database));
    $api->handle();
} catch (Throwable $exception) {
    error_log('[jiuin-php] bootstrap failure: ' . $exception->getMessage());
    http_response_code(503);
    header('Content-Type: application/json; charset=utf-8');
    echo json_encode(['code' => 503, 'message' => 'service is unavailable'], JSON_UNESCAPED_UNICODE);
}
