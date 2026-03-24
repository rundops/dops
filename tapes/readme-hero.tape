Output assets/demo.gif

Set Shell "zsh"
Set Width 1800
Set Height 1000
Set FontSize 18
Set Theme "Catppuccin Mocha"

# Launch dops
Type "DOPS_NO_ALT_SCREEN=1 ./bin/dops"
Enter
Sleep 2s

Screenshot assets/hero-startup.png

# Navigate to demo catalog → deploy-app
Down
Sleep 200ms
Down
Sleep 200ms
Down
Sleep 200ms
Down
Sleep 200ms
Down
Sleep 200ms
Down
Sleep 200ms
Down
Sleep 200ms
Down
Sleep 200ms
Down
Sleep 200ms
Down
Sleep 200ms
Down
Sleep 200ms
Down
Sleep 200ms
Down
Sleep 200ms
Down
Sleep 200ms
Down
Sleep 200ms
Down
Sleep 200ms
Down
Sleep 200ms
Down
Sleep 200ms
Down
Sleep 200ms
Down
Sleep 200ms
Down
Sleep 200ms
Down
Sleep 200ms
Down
Sleep 200ms
Down
Sleep 200ms
Down
Sleep 200ms

Screenshot assets/hero-sidebar.png

# Execute deploy-app
Enter
Sleep 1s

Screenshot assets/hero-wizard-select.png

# Select "staging" (default)
Down
Sleep 300ms
Enter
Sleep 500ms
Enter
Sleep 500ms

# Type version
Type "v1.2.3"
Sleep 500ms

Screenshot assets/hero-wizard-input.png

Enter
Sleep 500ms
Enter
Sleep 500ms

# Multi-select features: toggle logging, monitoring, tracing
Sleep 300ms
Space
Sleep 200ms
Down
Sleep 200ms
Space
Sleep 200ms
Down
Sleep 200ms
Space
Sleep 200ms

Screenshot assets/hero-wizard-multiselect.png

Enter
Sleep 500ms

# Secret field — press enter to keep saved value
Screenshot assets/hero-wizard-secret.png

Enter
Sleep 500ms

# Boolean dry_run — default No
Enter
Sleep 500ms

# Risk confirmation (high risk = y/N)
Screenshot assets/hero-confirm.png

Type "y"
Sleep 4s

Screenshot assets/hero-execution.png

# Tab to output, scroll
Tab
Sleep 300ms
Up
Sleep 200ms
Up
Sleep 200ms
Up
Sleep 200ms
Up
Sleep 200ms

Screenshot assets/hero-scroll.png

# Search
Type "/"
Sleep 300ms
Type "Deploy"
Sleep 500ms

Screenshot assets/hero-search.png

Escape
Sleep 300ms

# Help
Type "?"
Sleep 1s

Screenshot assets/hero-help.png

Escape
Sleep 300ms

Type "q"
Sleep 500ms
