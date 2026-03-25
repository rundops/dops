Output tapes/screenshots/p1-verify.gif

Set Shell "zsh"
Set Width 1200
Set Height 800
Set FontSize 14
Set Theme "Catppuccin Mocha"

Type "DOPS_NO_ALT_SCREEN=1 ./dops"
Enter
Sleep 2s

Screenshot tapes/screenshots/p1-verify-startup.png

# Navigate to a runbook and open wizard
Down
Sleep 300ms
Down
Sleep 300ms
Enter
Sleep 500ms

Screenshot tapes/screenshots/p1-verify-wizard.png

Escape
Sleep 500ms

# Tab to output pane
Tab
Sleep 300ms

Screenshot tapes/screenshots/p1-verify-output-focus.png

Type "q"
Sleep 500ms
