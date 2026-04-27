<?php
// Apache auto_prepend_file 진입점. v1 레거시 호환성 모듈을 순차 로딩한다.
require __DIR__ . '/_legacy_docroot.php';
require __DIR__ . '/_legacy_url_rewriter.php';
