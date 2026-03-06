package cmd

import (
	"fmt"
	"strings"
)

func renderInitScript(shell string, noCD bool) (string, error) {
	shell = strings.ToLower(shell)
	switch shell {
	case "bash", "zsh":
		return renderPOSIXInitScript(shell, noCD), nil
	case "fish":
		return renderFishInitScript(noCD), nil
	default:
		return "", fmt.Errorf("unsupported shell %q", shell)
	}
}

func renderPOSIXInitScript(shell string, noCD bool) string {
	header := fmt.Sprintf("# git-ho shell integration for %s", shell)
	if noCD {
		return header + "\n# Automatic cd is disabled because --nocd was specified.\n"
	}
	return header + "\n" + fmt.Sprintf(`git() {
  if [ "$1" != "ho" ]; then
    command git "$@"
    return $?
  fi

  shift
  local nocd_config nocd_flag output exit_code last_line should_cd
  nocd_config="$(command git config --bool --get ho.nocd 2>/dev/null || true)"
  nocd_flag=false
  for arg in "$@"; do
    if [ "$arg" = "--nocd" ]; then
      nocd_flag=true
      break
    fi
  done

  output="$(GIT_HO_SHELL_INTEGRATION=1 command git-ho "$@")"
  exit_code=$?
  if [ -n "$output" ]; then
    printf '%%s\n' "$output"
  fi
  if [ $exit_code -ne 0 ]; then
    return $exit_code
  fi

  if [ -z "$output" ]; then
    return 0
  fi

  last_line="$(printf '%%s\n' "$output" | tail -n 1)"
  if [ ! -d "$last_line" ]; then
    return 0
  fi

  should_cd=true
  if [ "$nocd_flag" = "true" ] || [ "$nocd_config" = "true" ]; then
    should_cd=false
  fi
  if [ "$should_cd" = "true" ]; then
    cd "$last_line" || return 1
  fi
}
`)
}

func renderFishInitScript(noCD bool) string {
	header := "# git-ho shell integration for fish"
	if noCD {
		return header + "\n# Automatic cd is disabled because --nocd was specified.\n"
	}
	return header + `
function git --wraps git
    if test (count $argv) -eq 0
        command git $argv
        return $status
    end
    if test "$argv[1]" != "ho"
        command git $argv
        return $status
    end

    set -e argv[1]
    set -l nocd_config (command git config --bool --get ho.nocd 2>/dev/null)
    set -l nocd_flag false
    for arg in $argv
        if test "$arg" = "--nocd"
            set nocd_flag true
            break
        end
    end

    set -lx GIT_HO_SHELL_INTEGRATION 1
    set -l output (command git-ho $argv)
    set -l status_code $status
    if test (count $output) -gt 0
        printf "%s\n" $output
    end
    if test $status_code -ne 0
        return $status_code
    end
    if test (count $output) -eq 0
        return 0
    end

    set -l last_line $output[-1]
    if not test -d "$last_line"
        return 0
    end
    if test "$nocd_flag" = "true" -o "$nocd_config" = "true"
        return 0
    end

    cd "$last_line"
end
`
}
