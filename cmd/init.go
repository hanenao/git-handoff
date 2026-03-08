package cmd

import (
	"fmt"
	"strings"
)

const bashCompletion = `
# git ho completion for bash
_git_ho() {
    local cur words cword
    _get_comp_words_by_ref -n =: cur words cword 2>/dev/null || {
        cur="${COMP_WORDS[COMP_CWORD]}"
        words=("${COMP_WORDS[@]}")
        cword=$COMP_CWORD
    }
    local args=("${words[@]:2}")
    local completions
    completions="$(command git-ho __complete "${args[@]}" 2>/dev/null | grep -v '^:' | cut -f1)"
    if declare -F __gitcomp_nl >/dev/null; then
        __gitcomp_nl "$completions"
        return
    fi
    COMPREPLY=($(compgen -W "$completions" -- "$cur"))
}

_git_ho_direct() {
    local cur words cword
    _get_comp_words_by_ref -n =: cur words cword 2>/dev/null || {
        cur="${COMP_WORDS[COMP_CWORD]}"
        words=("${COMP_WORDS[@]}")
        cword=$COMP_CWORD
    }
    COMPREPLY=()
    while IFS= read -r line; do
        COMPREPLY+=("$line")
    done < <(command git-ho __complete "${words[@]:1}" 2>/dev/null | grep -v '^:' | cut -f1)
}

complete -o nospace -F _git_ho_direct git-ho
`

const zshCompletion = `
# git ho completion for zsh with descriptions
_git-ho() {
    local -a completions
    local args=("${words[@]:1}")
    while IFS=$'\t' read -r comp desc; do
        [[ "$comp" == :* ]] && continue
        if [[ -n "$desc" ]]; then
            completions+=("${comp}:${desc}")
        else
            completions+=("${comp}")
        fi
    done < <(command git-ho __complete "${args[@]}" 2>/dev/null)
    _describe 'git-ho' completions
}

_git-ho-wrapper() {
    if (( CURRENT == 2 )); then
        _git
    elif [[ "${words[2]}" == "ho" ]]; then
        shift words
        (( CURRENT-- ))
        _git-ho
    else
        _git
    fi
}

if (( $+functions[compdef] )); then
    compdef _git-ho git-ho
    compdef _git-ho-wrapper git
fi
`

const bashGitWrapper = `
git() {
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
    printf '%s\n' "$output"
  fi
  if [ $exit_code -ne 0 ]; then
    return $exit_code
  fi

  if [ -z "$output" ]; then
    return 0
  fi

  last_line="$(printf '%s\n' "$output" | tail -n 1)"
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
`

const zshGitWrapper = `
git() {
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
    printf '%s\n' "$output"
  fi
  if [ $exit_code -ne 0 ]; then
    return $exit_code
  fi

  if [ -z "$output" ]; then
    return 0
  fi

  last_line="$(printf '%s\n' "$output" | tail -n 1)"
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
`

const fishGitWrapper = `
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

const fishCompletion = `
# git ho completion for fish
function __fish_git_ho_completions
    set -l cmd (commandline -opc)
    set -l args $cmd[3..]
    set -l cur (commandline -ct)
    command git-ho __complete $args "$cur" 2>/dev/null | string match -rv '^:'
end

function __fish_git_ho_direct_completions
    set -l cmd (commandline -opc)
    set -l args $cmd[2..]
    set -l cur (commandline -ct)
    command git-ho __complete $args "$cur" 2>/dev/null | string match -rv '^:'
end

function __fish_git_ho_needs_completion
    set -l cmd (commandline -opc)
    test (count $cmd) -ge 2 -a "$cmd[2]" = "ho"
end

complete -x -c git -n '__fish_git_ho_needs_completion' -a '(__fish_git_ho_completions)'
complete -x -c git-ho -a '(__fish_git_ho_direct_completions)'
`

func renderInitScript(shell string, noCD bool) (string, error) {
	shell = strings.ToLower(shell)
	switch shell {
	case "bash":
		return renderBashInitScript(noCD), nil
	case "zsh":
		return renderZshInitScript(noCD), nil
	case "fish":
		return renderFishInitScript(noCD), nil
	default:
		return "", fmt.Errorf("unsupported shell %q", shell)
	}
}

func renderBashInitScript(noCD bool) string {
	return renderShellInitScript("# git-ho shell integration for bash", bashGitWrapper, bashCompletion, noCD)
}

func renderZshInitScript(noCD bool) string {
	return renderShellInitScript("# git-ho shell integration for zsh", zshGitWrapper, zshCompletion, noCD)
}

func renderFishInitScript(noCD bool) string {
	return renderShellInitScript("# git-ho shell integration for fish", fishGitWrapper, fishCompletion, noCD)
}

func renderShellInitScript(header, wrapper, completion string, noCD bool) string {
	var builder strings.Builder
	builder.WriteString(header)
	builder.WriteString("\n")
	if noCD {
		builder.WriteString("# Automatic cd is disabled because --nocd was specified.\n")
	} else {
		builder.WriteString(wrapper)
		if !strings.HasSuffix(wrapper, "\n") {
			builder.WriteString("\n")
		}
	}
	builder.WriteString(completion)
	if !strings.HasSuffix(completion, "\n") {
		builder.WriteString("\n")
	}
	return builder.String()
}
