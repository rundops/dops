Output tapes/screenshots/outer-border.gif

Set Shell "zsh"
Set Width 1400
Set Height 900
Set FontSize 16
Set Theme "Catppuccin Mocha"

Type "DOPS_NO_ALT_SCREEN=1 ./dops"
Enter
Sleep 2s
Screenshot tapes/screenshots/outer-border-01-before.png

Enter
Sleep 1s
Type "google.com"
Sleep 300ms
Enter
Sleep 4s
Screenshot tapes/screenshots/outer-border-02-after.png

Tab
Sleep 300ms
Up
Sleep 300ms
Up
Sleep 300ms
Up
Sleep 300ms
Screenshot tapes/screenshots/outer-border-03-scrolled.png

Type "q"
Sleep 500ms
