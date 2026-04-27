<?php
// v1 PHP 가 사용하는 $_SERVER['DOCUMENT_ROOT'] 를 v1 디렉토리로 보정.
// Apache 의 실제 DocumentRoot 는 v2 SPA(/var/www/app) 이므로,
// v1 코드의 require_once $_SERVER['DOCUMENT_ROOT'].'/_sys/...' 패턴이
// 올바른 경로(/var/www/html)를 찾도록 강제한다.
$_SERVER['DOCUMENT_ROOT'] = '/var/www/html';
chdir('/var/www/html');
