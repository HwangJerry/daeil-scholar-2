# MariaDB 10.1 인덱스 호환성 배포 보완

2026-09-08. 운영 058은 WEO_MEMBER_PUSH의 utf8 VARCHAR(300) 기본키가
기존 InnoDB 767바이트 한도를 초과하여 실패했다. 055–057만 이력에 기록됐으며
058에서 앞서 완료된 다섯 테이블의 InnoDB 변환은 유지된 상태다.

058의 남은 MyISAM 변환에 ROW_FORMAT=DYNAMIC을 지정하여 기본키 길이를 유지한다.
이미 InnoDB인 테이블은 다시 변환하지 않는다. 이미 058을 적용한 다른 환경에는
변경된 migration을 강제로 재적용하지 않는다. 해시 불일치 사전 검사는 유지한다.

배포 사전 검사는 058이 미적용이면 Barracuda, large_prefix=ON,
file_per_table=ON, 16KB 페이지를 요구한다. 미충족이면 서비스 중지 전에 차단한다.
이 후보는 검증한 MariaDB 10.1 설정을 대상으로 한다.

관리자 권한으로 적용할 실행 중 설정:

```sql
SET GLOBAL innodb_file_format='Barracuda';
SET GLOBAL innodb_large_prefix=ON;
```

다음 시작에도 유지할 별도 서버 설정(아직 운영에 작성하지 않음):

```ini
[mysqld]
innodb_file_format=Barracuda
innodb_large_prefix=ON
```

기존 file_per_table=ON, 메모리 크기·접속 수는 변경하지 않는다. DB 재시작이나
앱 DB 계정 권한 확대는 필요하지 않으며 시행하지 않는다. 테이블 변환·배포는
기존 서비스 중지/백업 절차로 수행하고 탈퇴/보관 만료 기능은 비활성으로 유지한다.
변환 후 설정값만 낮추는 것은 변환된 테이블의 저장 형식을 되돌리는 작업이 아니다.

검증:
- MariaDB 10.1.38 로컬 합성 자료에서 원래 오류 1071 재현.
- 실행 중 설정 변경 후 수정 058을 두 번 실행해 모두 통과.
- 300자 한국어 PUSH_ID와 ASCII 토큰, 회원번호의 전체 값 보존.
- 기본키 prefix 제한 없음(SHOW INDEX Sub_part=NULL), InnoDB/Dynamic 확인.
- 이미 변환된 InnoDB 테이블의 자료 유지.
- Python 배포 테스트 18개 통과. 잘못된 설정에서 서비스 중지 명령 미실행 확인.

운영 DB 관리자 인증은 아직 확보하지 못했다. 표준 인증 파일과 프로젝트의 명시적
관리자 환경 키를 확인했으나 찾지 못했다. 비밀번호 추측이나 권한 우회는 하지 않는다.
