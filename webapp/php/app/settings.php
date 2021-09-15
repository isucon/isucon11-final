<?php

declare(strict_types=1);

use App\Application\Settings\Settings;
use App\Application\Settings\SettingsInterface;
use DI\ContainerBuilder;
use Monolog\Logger;

return function (ContainerBuilder $containerBuilder) {

    // Global Settings Object
    $containerBuilder->addDefinitions([
        SettingsInterface::class => function () {
            return new Settings([
                'displayErrorDetails' => getenv('DEBUG') === 'true', // Should be set to false in production
                'logError'            => true,
                'logErrorDetails'     => true,
                'logger' => [
                    'name' => 'slim-app',
                    'path' => 'php://stdout',
                    'level' => Logger::DEBUG,
                ],
                'database' => [
                    'host' => getenv('MYSQL_HOSTNAME') ?: '127.0.0.1',
                    'port' => getenv('MYSQL_PORT') ?: '3306',
                    'database' => getenv('MYSQL_DATABASE') ?: 'isucholar',
                    'user' => getenv('MYSQL_USER') ?: 'isucon',
                    'password' => getenv('MYSQL_PASS') ?: 'isucon',
                ],
                'session' => [
                    'lifetime' => '3600 seconds',
                    'name' => 'isucholar_php',
                ],
            ]);
        }
    ]);
};
