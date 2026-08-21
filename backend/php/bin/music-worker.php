#!/usr/bin/env php
<?php
declare(strict_types=1);

use Jiuin\Config;
use Jiuin\Database;
use Jiuin\MusicStore;

require dirname(__DIR__) . '/src/Config.php';
require dirname(__DIR__) . '/src/Database.php';
require dirname(__DIR__) . '/src/MusicStore.php';

$config = Config::fromEnvironment();
$config->ensureStorage();
$store = new MusicStore($config, Database::open($config));
$workerId = 'php-' . gethostname() . '-' . getmypid();

while (true) {
    try {
        $worked = $store->processOne($workerId);
        if (!$worked) {
            sleep($config->workerIntervalSeconds);
        }
    } catch (Throwable $exception) {
        error_log('[jiuin-php-worker] ' . $exception->getMessage());
        sleep($config->workerIntervalSeconds);
    }
}
