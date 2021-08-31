<?php

declare(strict_types=1);

use App\Application\Middleware\AccessLog;
use Slim\App;
use Slim\Middleware\Session;

return function (App $app) {
    $app->add(AccessLog::class);
    $app->add(Session::class);
};
