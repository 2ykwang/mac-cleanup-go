<div align="center">
  <h1>mac-cleanup-go</h1>
  <p>TUI 기반으로 삭제할 항목을 선택하여 macOS 캐시, 로그, 임시 파일을 정리합니다.</p>
</div>

<p align="center">
  <a href="https://github.com/2ykwang/mac-cleanup-go/releases"><img src="https://img.shields.io/github/v/release/2ykwang/mac-cleanup-go" alt="GitHub Release"></a>
  <a href="https://goreportcard.com/report/github.com/2ykwang/mac-cleanup-go"><img src="https://goreportcard.com/badge/github.com/2ykwang/mac-cleanup-go" alt="Go Report Card"></a>
  <a href="https://github.com/2ykwang/mac-cleanup-go/actions/workflows/test.yml"><img src="https://github.com/2ykwang/mac-cleanup-go/actions/workflows/test.yml/badge.svg" alt="CI"></a>
  <a href="https://codecov.io/gh/2ykwang/mac-cleanup-go"><img src="https://codecov.io/gh/2ykwang/mac-cleanup-go/graph/badge.svg?token=ecH3KP0piI" alt="codecov"/></a>
  <a href="https://golangci-lint.run/"><img src="https://img.shields.io/badge/linted%20by-golangci--lint-brightgreen" alt="golangci-lint"></a>
</p>

<p align="center">
  <a href="README.md">English</a> | <a href="README_KO.md">한국어</a>
</p>

## 개요

- 삭제될 항목을 미리보기하고 제외할 수 있습니다.
- 기본은 휴지통으로 이동하고, '휴지통' 카테고리만 영구 삭제합니다.
- risky 항목은 기본 제외, manual은 가이드만 표시합니다.
- 범위: 캐시/로그/임시 파일과 일부 앱 데이터. 시스템 최적화/앱 제거는 하지 않습니다.

## 빠른 시작

Homebrew로 설치:

```bash
brew install 2ykwang/2ykwang/mac-cleanup-go
```

실행:

```bash
mac-cleanup
mac-cleanup --update   # Homebrew로 업데이트
```

> 팁: 휴지통과 제한된 위치를 정리하려면 터미널에 전체 디스크 접근 권한을 부여하세요.  
> 시스템 설정 -> 개인정보 보호 및 보안 -> 전체 디스크 접근

![demo](assets/demo.gif)

## 무엇을 하는지

- 앱/도구별 알려진 캐시, 로그, 임시 경로를 병렬로 스캔합니다.
- 미리보기에서 항목을 제외할 수 있습니다.
- 영향도 라벨(safe, moderate, risky)을 표시합니다.
- Homebrew/Docker 명령 결과와 마지막 수정 시각 기준으로 동작하는 내장 스캔이 있습니다.

> 참고: risky 카테고리는 기본 선택되지만 모든 항목이 제외된 상태로 시작합니다.
> 삭제하려면 미리보기 페이지에서 항목을 직접 포함해야 합니다.

## 영향도 분류 기준

- safe: 자동 재생성되는 캐시/로그.
- moderate: 재다운로드 또는 재로그인이 필요할 수 있음.
- risky: 사용자 데이터가 포함될 수 있음; 항목 기본 제외.
- manual: 자동 삭제 없이 앱 가이드만 표시.

## 대상 요약

- 전체 대상: 107개.
- 그룹: System 7, Browsers 10, Development 35, Applications 52, Storage 3.
- 처리 방법: trash 101, permanent 1, builtin 3, manual 2.
- builtin: homebrew, docker, old-downloads (고정 경로가 아닌 명령 결과/마지막 수정 시각 기준 스캔).
- manual: telegram, kakaotalk (자동 삭제하지 않고, 채팅 데이터 등 대용량 항목 안내 목적).

## 사용 참고

- 전체 디스크 접근 권한이 있으면 제한된 위치도 스캔/정리할 수 있습니다.
- 버전 확인: `mac-cleanup --version`.

<details>
<summary><strong>키 바인딩</strong></summary>

목록 화면:

- `Up`/`Down` 또는 `k`/`j`: 이동
- `Space`: 카테고리 선택
- `a`: 전체 선택, `d`: 전체 해제
- `Enter` 또는 `p`: 미리보기
- `?`: 도움말, `q`: 종료

미리보기 화면:

- `Up`/`Down` 또는 `k`/`j`: 이동
- `h`/`l`: 이전/다음 카테고리
- `Space`: 제외 토글
- `Enter`: 디렉터리 드릴다운
- `/`: 검색, `s`: 정렬, `o`: Finder 열기
- `a`: 전체 포함, `d`: 전체 제외
- `y`: 삭제(확인 포함), `esc`: 뒤로

확인 화면:

- `y` 또는 `Enter`: 확인
- `n` 또는 `esc`: 취소

</details>

## 대안

- [mac-cleanup-py](https://github.com/mac-cleanup/mac-cleanup-py) - Python cleanup script for macOS
- [Mole](https://github.com/tw93/Mole) - Deep clean and optimize your Mac

## License

MIT
