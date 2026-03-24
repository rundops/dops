Output tapes/gifs/stress-test.gif

Set Shell "zsh"
Set Width 1400
Set Height 900
Set FontSize 16
Set Theme "TokyoNight"

Type "DOPS_NO_ALT_SCREEN=1 ./bin/dops"
Enter
Sleep 2s

# Test 1: echo (simple string param)
Screenshot tapes/screenshots/stress-01-startup.png
Enter
Sleep 1s
Type "hello stress test"
Enter
Sleep 500ms
# Save prompt — say no
Type "n"
Sleep 2s
Screenshot tapes/screenshots/stress-02-echo.png

# Test 2: Navigate to generate-password (integer + select params)
Down
Sleep 200ms
Down
Sleep 200ms
Down
Sleep 200ms
Down
Sleep 200ms
Enter
Sleep 1s

# Integer: length
Screenshot tapes/screenshots/stress-03-integer.png
Enter
Sleep 500ms
Type "n"
Sleep 300ms

# Select: charset
Screenshot tapes/screenshots/stress-04-select.png
Down
Sleep 200ms
Enter
Sleep 500ms
Type "n"
Sleep 2s
Screenshot tapes/screenshots/stress-05-password-output.png

# Test 3: Navigate to encode-base64 (string + boolean params)
Down
Sleep 200ms
Enter
Sleep 1s

# String input
Type "Hello World"
Enter
Sleep 500ms
Type "n"
Sleep 300ms

# Boolean toggle
Screenshot tapes/screenshots/stress-06-boolean.png
Enter
Sleep 500ms
Type "n"
Sleep 2s
Screenshot tapes/screenshots/stress-07-base64-output.png

# Test 4: Navigate to infra.dns-lookup (string + select)
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
Enter
Sleep 1s
Type "google.com"
Enter
Sleep 500ms
Type "n"
Sleep 300ms

# Select record type
Screenshot tapes/screenshots/stress-08-dns-select.png
Enter
Sleep 500ms
Type "n"
Sleep 2s
Screenshot tapes/screenshots/stress-09-dns-output.png

# Test 5: Navigate to demo.audit-permissions (multi_select + string + boolean)
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
Enter
Sleep 1s

# Multi-select
Screenshot tapes/screenshots/stress-10-multiselect.png
Space
Sleep 200ms
Down
Sleep 200ms
Space
Sleep 200ms
Enter
Sleep 500ms
Type "n"
Sleep 300ms

# Account string
Type "123456789012"
Enter
Sleep 500ms
Type "n"
Sleep 300ms

# Verbose boolean
Enter
Sleep 500ms
Type "n"
Sleep 2s
Screenshot tapes/screenshots/stress-11-audit-output.png

# Test 6: Scroll and search in output
Tab
Sleep 300ms
Up
Sleep 200ms
Up
Sleep 200ms
Up
Sleep 200ms
Screenshot tapes/screenshots/stress-12-scroll.png

Type "/"
Sleep 200ms
Type "findings"
Sleep 300ms
Screenshot tapes/screenshots/stress-13-search.png
Escape
Sleep 300ms

# Test 7: Help overlay
Type "?"
Sleep 1s
Screenshot tapes/screenshots/stress-14-help.png
Escape
Sleep 300ms

Type "q"
Sleep 500ms
