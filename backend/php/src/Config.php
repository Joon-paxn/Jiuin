<?php
declare(strict_types=1);

namespace Jiuin;

use RuntimeException;

final class Config
{
    /** @param list<array{name:string,url:string,description:string}> $externalLinks
     *  @param list<array{name:string,url:string,priority:int,cachePolicy:string}> $resources
     *  @param list<string> $backgroundUrls */
    public function __construct(
        public readonly string $storageDir,
        public readonly string $databasePath,
        public readonly string $ffmpegPath,
        public readonly string $ffprobePath,
        public readonly string $fullBitrate,
        public readonly string $liteBitrate,
        public readonly string $outputCodec,
        public readonly string $adminToken,
        public readonly string $serviceToken,
        public readonly string $siteName,
        public readonly string $siteProject,
        public readonly string $siteDomain,
        public readonly array $externalLinks,
        public readonly array $resources,
        public readonly array $backgroundUrls,
        public readonly int $processingLeaseSeconds,
        public readonly int $workerIntervalSeconds,
    ) {}

    public static function fromEnvironment(): self
    {
        $storage = self::env('JIUIN_STORAGE_DIR', '/var/lib/jiuin/music');
        $links = self::jsonList('JIUIN_EXTERNAL_LINKS_JSON');
        $resources = self::jsonList('JIUIN_RESOURCE_MANIFEST_JSON');
        $backgrounds = self::jsonList('JIUIN_BACKGROUND_URLS_JSON');
        self::validateConfiguredURLs($links, 'external link');
        self::validateConfiguredURLs($resources, 'resource');
        self::validateConfiguredURLs($backgrounds, 'background');
        return new self(
            $storage,
            self::env('JIUIN_DATABASE_PATH', $storage . '/music.db'),
            self::env('JIUIN_FFMPEG_PATH', 'ffmpeg'),
            self::env('JIUIN_FFPROBE_PATH', 'ffprobe'),
            self::env('JIUIN_MUSIC_FULL_BITRATE', '320k'),
            self::env('JIUIN_MUSIC_LITE_BITRATE', '128k'),
            self::env('JIUIN_MUSIC_OUTPUT_CODEC', 'libmp3lame'),
            self::env('JIUIN_MUSIC_ADMIN_TOKEN', ''),
            self::env('JIUIN_SHARED_SERVICE_TOKEN', ''),
            self::env('JIUIN_SITE_NAME', '霁雪居'),
            self::env('JIUIN_SITE_PROJECT', 'Jiuin'),
            self::env('JIUIN_SITE_DOMAIN', 'jiuin.cn'),
            $links,
            $resources,
            $backgrounds,
            // The lease must cover normal FFmpeg work. An expired lease could
            // let the PHP and Go workers process the same task concurrently.
            self::positiveInt('JIUIN_MUSIC_PROCESSING_LEASE_SECONDS', 7200),
            self::positiveInt('JIUIN_MUSIC_WORKER_INTERVAL_SECONDS', 2),
        );
    }

    public function ensureStorage(): void
    {
        foreach (['', 'tmp', 'original', 'full', 'lite', 'covers'] as $child) {
            $path = $this->storageDir . ($child === '' ? '' : '/' . $child);
            if (!is_dir($path) && !mkdir($path, 0750, true) && !is_dir($path)) {
                throw new RuntimeException('Cannot create storage directory: ' . $path);
            }
        }
    }

    private static function env(string $name, string $fallback): string
    {
        $value = getenv($name);
        return is_string($value) && trim($value) !== '' ? trim($value) : $fallback;
    }

    private static function positiveInt(string $name, int $fallback): int
    {
        $value = filter_var(getenv($name), FILTER_VALIDATE_INT, ['options' => ['min_range' => 1]]);
        return is_int($value) ? $value : $fallback;
    }

    /** @return list<mixed> */
    private static function jsonList(string $name): array
    {
        $value = self::env($name, '[]');
        try {
            $parsed = json_decode($value, true, 512, JSON_THROW_ON_ERROR);
        } catch (\JsonException $exception) {
            throw new RuntimeException($name . ' must be a JSON array', 0, $exception);
        }
        if (!is_array($parsed) || !array_is_list($parsed)) {
            throw new RuntimeException($name . ' must be a JSON array');
        }
        return $parsed;
    }

    /** @param list<mixed> $items */
    private static function validateConfiguredURLs(array $items, string $kind): void
    {
        foreach ($items as $item) {
            $value = is_array($item) ? ($item['url'] ?? null) : $item;
            if (!is_string($value) || !self::isPublicURL($value)) {
                throw new RuntimeException($kind . ' URL must be root-relative or public HTTPS');
            }
        }
    }

    private static function isPublicURL(string $value): bool
    {
        if ($value === '' || str_starts_with($value, '//')) return false;
        if (str_starts_with($value, '/')) return true;
        $parts = parse_url($value);
        if (!is_array($parts) || ($parts['scheme'] ?? '') !== 'https' || isset($parts['user']) || !is_string($parts['host'] ?? null)) return false;
        $host = strtolower($parts['host']);
        return $host !== 'localhost' && $host !== 'bkgapi.jiuin.cn' && !str_ends_with($host, '.localhost') && filter_var($host, FILTER_VALIDATE_IP) === false;
    }
}
