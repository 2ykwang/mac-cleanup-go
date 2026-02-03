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
  <a href="../README.md">English</a> | <a href="README_KO.md">한국어</a>
</p>

## 개요

- 용량을 불필요하게 차지하는 항목을 직접 선택하여 삭제할 수 있습니다.
- 기본은 휴지통으로 이동하고, '휴지통' 카테고리만 영구 삭제합니다.
- risky 카테고리는 기본 미선택이며, 선택해도 항목은 자동 제외됩니다. (삭제하려면 미리보기에서 포함해야 합니다.)
- manual 카테고리는 삭제 가이드만 표시합니다.
- 작업 범위: 캐시/로그/임시 파일과 일부 앱 데이터. 시스템 최적화/앱 제거는 하지 않습니다.

![demo](result_view.png)

## 빠른 시작

**1) 설치**

```bash
brew install mac-cleanup-go
```

또는 [GitHub Releases](https://github.com/2ykwang/mac-cleanup-go/releases)에서 내려받아 실행하세요.

**2) (선택) 전체 디스크 접근 (휴지통/제한 경로 정리에 필요)**
시스템 설정 -> 개인정보 보호 및 보안 -> 전체 디스크 접근 -> Terminal 추가

**3) 실행**

```bash
mac-cleanup
```

팁: Enter로 미리보기, y로 삭제 진행. 키 바인딩은 ? 키로 확인하세요.

![demo](demo.gif)

- 업데이트: `brew upgrade mac-cleanup-go` 또는 `mac-cleanup --update`.
- 삭제: `brew uninstall mac-cleanup-go`.
- 디버그: `mac-cleanup --debug`로 로그 저장 (`~/.config/mac-cleanup-go/debug.log`).

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

## CLI 모드

명령줄에서 대상을 설정하고 정리를 실행합니다.

```bash
mac-cleanup --select                   # 정리 대상 설정
mac-cleanup --clean --dry-run          # 정리 리포트 미리보기
mac-cleanup --clean                    # 정리 실행
```

명령줄 정리에 대한 자세한 내용은 아래 예시를 참고하세요.

<details>
<summary><strong>실행 예시</strong></summary>

**1) 대상 선택**

```
$ mac-cleanup --select

Select cleanup targets                  ● safe  ○ moderate
─────────────────────────────────────────────────────────
        Name                                      Size
  [ ] ● Trash                                      0 B
  [✓] ○ App Caches                              3.2 GB
  [✓] ○ System Logs                           259.7 MB
▸ [✓] ● Go Build Cache                        845.0 MB
  [✓] ○ Docker                                  2.8 GB
  [✓] ● Homebrew Cache                          1.5 GB
  [ ] ● Chrome Cache                               0 B
─────────────────────────────────────────────────────────
Selected: 5
↑/↓ Move  space Select  s Save  ? Help  q Cancel
```

**2) 미리보기 / 정리**

```
$ mac-cleanup --clean --dry-run

Dry Run Report
--------------
Mode: Dry Run

Summary                             Highlights
Freed (dry-run): 8.6 GB             1. App Caches - 3.2 GB (523 items)
                                    2. Docker - 2.8 GB (12 items)
                                    3. Homebrew Cache - 1.5 GB (34 items)

Details
STATUS  CATEGORY              ITEMS        SIZE
OK      App Caches              523      3.2 GB
OK      Docker                   12      2.8 GB
OK      Homebrew Cache           34      1.5 GB
OK      Go Build Cache           89    845.0 MB
OK      System Logs              67    259.7 MB
```

</details>

## 작동 방식 및 안전 정책

- 앱/도구별 알려진 캐시, 로그, 임시 경로를 병렬로 스캔합니다.
- 삭제 전 미리보기에서 항목을 제외할 수 있습니다.
- 영향도 라벨(safe, moderate, risky, manual)을 표시합니다.
- SIP 보호 경로는 스캔/정리 대상에서 제외됩니다.
- Homebrew, Docker, 오래된 다운로드 파일에 대한 전용 스캐너가 있습니다 (brew/docker 출력 또는 마지막 수정 시각 기준).

## 영향도 분류 기준

- safe: 자동 재생성되는 캐시/로그.
- moderate: 재다운로드 또는 재로그인이 필요할 수 있음.
- risky: 사용자 데이터가 포함될 수 있음; 항목 기본 제외.
- manual: 자동 삭제 없이 앱 가이드만 표시.

## 대상 (v1.3.6 기준)

- 전체 대상: 107개.
- 그룹: System 7, Browsers 10, Development 35, Applications 52, Storage 3.
- 처리 방법: trash 101, permanent 1, builtin 3, manual 2.
- builtin: homebrew, docker, old-downloads (고정 경로가 아닌 명령 결과/마지막 수정 시각 기준 스캔).
- manual: telegram, kakaotalk (자동 삭제하지 않고, 채팅 데이터 등 대용량 항목 안내 목적).
- 수치는 릴리스 기준이며 변동될 수 있습니다.

## 대안 오픈소스

- [mac-cleanup-py](https://github.com/mac-cleanup/mac-cleanup-py) - Python cleanup script for macOS
- [Mole](https://github.com/tw93/Mole) - Deep clean and optimize your Mac

## License

MIT
