<?php
// Maintenance sentinel gate for every dynamic legacy PHP request.
$maintenanceSentinel = getenv('ALUMNI_MAINTENANCE_SENTINEL');
if ($maintenanceSentinel === false || trim($maintenanceSentinel) === '') {
    $maintenanceSentinel = '/run/alumni/maintenance';
}

$sentinelParent = dirname($maintenanceSentinel);
$sentinelStateInvalid = !is_dir($sentinelParent) || !is_readable($sentinelParent);
$maintenanceActive = @lstat($maintenanceSentinel) !== false;

if ($maintenanceActive || $sentinelStateInvalid) {
    http_response_code(503);
    header('Content-Type: application/json; charset=utf-8');
    header('Retry-After: 60');
    echo '{"code":"MAINTENANCE_MODE","message":"Legacy writes are temporarily unavailable"}';
    exit;
}
