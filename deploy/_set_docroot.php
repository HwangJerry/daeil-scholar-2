<?php
// v1 레거시 사이트(/old/ 별칭) 응답에서 루트 기준 절대 URL을 /old/ 로 재작성한다.
// Apache auto_prepend_file 로 모든 PHP 요청에 자동 적용된다.

ob_start(function (string $html): string {
    $prefixes = '_sys|_gateway|_module|_dialog|_skin|_logAPI|_ajax|images|upload|tpl|pc|mob|memo|news|biz|notice|event|relay|schoolmate|suggest|qna|mypage|login|joinus|scholarship|ranking|donation|dona|info|stat|service';

    // (a) HTML 속성: href/src/action="/<prefix>..." -> "/old/<prefix>..."
    $html = preg_replace(
        '#((?:href|src|action)\s*=\s*["\'])/(' . $prefixes . ')(?=[/"\'])#i',
        '$1/old/$2',
        $html
    );

    // (b) 인라인 JS 문자열 리터럴: "/<prefix>/..." -> "/old/<prefix>/..."
    $html = preg_replace(
        '#(["\'])/(' . $prefixes . ')(/)#',
        '$1/old/$2$3',
        $html
    );

    return $html;
});
