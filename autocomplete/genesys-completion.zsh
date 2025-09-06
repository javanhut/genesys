#compdef genesys

# Zsh completion for genesys
function _genesys {
    local context curcontext="$curcontext" state line
    typeset -A opt_args

    local ret=1

    _arguments -C \
        '--help[Show help information]' \
        '--version[Show version information]' \
        '--verbose[Enable verbose output]' \
        '--debug[Enable debug output]' \
        '1: :->cmds' \
        '*::arg:->args' && ret=0

    case $state in
    cmds)
        _values "genesys command" \
            'interact[Interactive resource creation wizard]' \
            'execute[Execute a configuration file]' \
            'discover[Discover existing cloud resources]' \
            'config[Manage Genesys configuration]' \
            'version[Show version information]' \
            'help[Show help information]'
        ret=0
        ;;
    args)
        case $line[1] in
        execute)
            _arguments \
                '--dry-run[Preview changes without applying]' \
                '--force[Force execution without confirmation]' \
                '--output=[Output format]:format:(json yaml table)' \
                '--parallel[Execute resources in parallel]' \
                '*:file:_files -g "*.{yaml,yml,toml}"' && ret=0
            ;;
        discover)
            if (( CURRENT == 2 )); then
                _values "provider" aws gcp azure tencent && ret=0
            elif (( CURRENT == 3 )); then
                _values "resource type" compute storage network database serverless all && ret=0
            else
                _arguments \
                    '--output=[Output file]:file:_files' \
                    '--format=[Output format]:format:(json yaml table)' \
                    '--filter=[Filter resources]' && ret=0
            fi
            ;;
        config)
            if (( CURRENT == 2 )); then
                _values "config command" setup list show validate && ret=0
            elif (( CURRENT == 3 )); then
                case $line[2] in
                setup|show|validate)
                    _values "provider" aws gcp azure tencent && ret=0
                    ;;
                esac
            else
                _arguments \
                    '--global[Use global configuration]' \
                    '--show-path[Show configuration file path]' && ret=0
            fi
            ;;
        interact|version|help)
            # No additional arguments
            ;;
        esac
        ;;
    esac

    return ret
}