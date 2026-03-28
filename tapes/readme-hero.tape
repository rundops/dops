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

# Search for deploy-app using filter
Type "/"
Sleep 500ms
Type "deploy-app"
Sleep 1s

Screenshot assets/hero-sidebar.png

# Confirm filter, navigate to deploy-app, execute
Enter
Sleep 500ms
Down
Sleep 500ms
Enter
Sleep 1500ms

Screenshot assets/hero-wizard-select.png

# Select "staging" (default)
Down
Sleep 500ms
Enter
Sleep 1s
Enter
Sleep 1s

# Type version
Type "v1.2.3"
Sleep 1500ms

Screenshot assets/hero-wizard-input.png

Enter
Sleep 1s
Enter
Sleep 1s

# Multi-select features: toggle logging, monitoring, tracing
Sleep 500ms
Space
Sleep 300ms
Down
Sleep 300ms
Space
Sleep 300ms
Down
Sleep 300ms
Space
Sleep 500ms

Screenshot assets/hero-wizard-multiselect.png

Enter
Sleep 1s

# Save prompt for features — decline
Enter
Sleep 1s

# Secret field — type a fake API token
Type "sk-dops-a1b2c3d4e5f6"
Sleep 1500ms

Screenshot assets/hero-wizard-secret.png

Enter
Sleep 1s
Enter
Sleep 1s

# Boolean dry_run — default No
Enter
Sleep 1s

# Risk confirmation (high risk = y/N) — toggle to Yes first
Left
Sleep 500ms

Screenshot assets/hero-confirm.png

Enter
Sleep 6s

Screenshot assets/hero-execution.png

# Tab to output, scroll
Tab
Sleep 500ms
Up
Sleep 300ms
Up
Sleep 300ms
Up
Sleep 300ms
Up
Sleep 300ms

Screenshot assets/hero-scroll.png

# Search
Type "/"
Sleep 500ms
Type "Deploy"
Sleep 2s

Screenshot assets/hero-search.png

Escape
Sleep 500ms

# Help
Type "?"
Sleep 1500ms

Screenshot assets/hero-help.png

Escape
Sleep 500ms

Type "q"
Sleep 500ms
