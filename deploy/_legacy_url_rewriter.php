<?php
// v1 응답 본문의 루트 기준 절대 URL(/_sys/, /_gateway/ 등) 을 /old/ 로 재작성한다.
// v1 은 도메인 루트에 배포된다는 가정으로 작성되었으나 v2 SPA 가 루트를 점유하므로
// 출력 단계에서 /old/ 프리픽스를 부여해 자산이 정상 로드되도록 한다.

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
