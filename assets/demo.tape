# GOSCII demo: hub -> onboarding (decode, repair, fault) -> AI fault mission
# -> comms with GOSCII -> station.
#
# Prereqs:
#   go build -o goscii .                              # local binary in repo root
#   go install github.com/charmbracelet/vhs@latest    # v0.8+ for Wait, plus ffmpeg and ttyd
#   export ANTHROPIC_API_KEY=...                      # the AI part forges a real mission
#
# Render:
#   vhs assets/demo.tape                              # writes assets/demo.gif
#
# Progress lives in a throwaway XDG_DATA_HOME, so the run starts from the same
# checkpoint every time and never touches your own save.

Output assets/demo.gif

Set Shell "bash"
Set FontSize 18
Set Width 1280
Set Height 920
Set Padding 20
Set Theme "Catppuccin Mocha"
Set TypingSpeed 70ms

Hide
Type `export XDG_DATA_HOME=$(mktemp -d) && mkdir -p $XDG_DATA_HOME/goscii && echo '{"adventure_checkpoint":"adventures/onboarding/03_types_decode"}' > $XDG_DATA_HOME/goscii/progress.json && clear`
Enter
Show

# --- home hub; with a key wired the cursor goes Signal -> Mission -> Adventure ---
Type "./goscii start" Enter
Sleep 2.5s
Down
Sleep 300ms
Down
Sleep 600ms
Enter
Sleep 4s          # Echo Check: a decode mission, read the program

# --- decode: type the predicted output ---
Type "3" Enter
Type "1" Enter
Type "3.5" Enter
Type "0"
Sleep 800ms
Ctrl+R
Sleep 2.5s

# --- repair: write the missing function (parens and braces auto-pair) ---
Ctrl+N
Sleep 3s
Type `func kelvin(c float64) float64 { return c + 273.15 }`
Sleep 800ms
Ctrl+R
Sleep 2.5s

# --- fault: the code is there and lies; run it, then fix the conversion ---
Ctrl+N
Sleep 3.5s
Ctrl+R
Sleep 2.5s        # got "22", the target is 22.25
Up
End
Backspace 15
Sleep 400ms
Type `(sum) / float64(len(xs))`
Sleep 800ms
Ctrl+R
Sleep 2.5s

# --- AI mission: Slices, kind Fault ---
Escape
Sleep 1s
Up                # Adventure -> Mission
Sleep 400ms
Enter
Sleep 800ms
Down 3            # variables -> functions -> control flow -> slices
Sleep 600ms
Enter
Sleep 1s
Down 2            # Mixed -> Repair -> Fault
Sleep 800ms
Enter
Sleep 5s          # forge stages: transmitting, test-firing, recalibrating
Hide
Wait+Screen@3m /SIGNAL/
Show
Sleep 4s          # the forged mission, already test-fired

# run the planted fault, then ask GOSCII to read it
Ctrl+R
Sleep 3s
Ctrl+G
Sleep 800ms
Enter             # empty line: GOSCII reads the last fault
Sleep 3s
Hide
Sleep 25s
Show
Sleep 7s          # GOSCII's reading of the fault

# --- station: the onboarding repairs show on the systems ---
Escape
Sleep 1s
Down 4            # Mission -> Adventure -> Difficulty -> Archive -> Station
Sleep 600ms
Enter
Sleep 5s
