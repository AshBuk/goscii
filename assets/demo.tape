# GOSCII demo — full first-contact run:
# launch -> progress log -> adventure -> solve level 1 (hints + run) -> level 2.
#
# Prereqs:
#   go build -o goscii .            # local binary in repo root
#   go install github.com/charmbracelet/vhs@latest   # + ffmpeg, ttyd
#
# Render:
#   vhs assets/demo.tape            # writes assets/demo.gif

Output assets/demo.gif

Set Shell "bash"
Set FontSize 18
Set Width 1280
Set Height 920
Set Padding 20
Set Theme "Catppuccin Mocha"
Set TypingSpeed 90ms

Hide
Type "clear" Enter
Show

# --- launch the home hub ---
Type "./goscii start" Enter
Sleep 2.5s

# --- peek the progress log: down to the bottom (Quit), one up = Progress.
# Robust whether or not the Signal row is wired (which shifts the menu). ---
Down
Down
Down
Down
Down
Up
Sleep 800ms
Enter
Sleep 3.5s        # bars sweep up to the completion counts
Escape
Sleep 800ms

# --- into the adventure: Progress -> Difficulty -> Adventure ---
Up
Up
Sleep 800ms
Enter
Sleep 2.5s        # Cold Boot — let the story breathe

# --- cycle every hint ---
Ctrl+H
Sleep 1s
Ctrl+H
Sleep 1s
Ctrl+H
Sleep 1s

# --- solve it (quotes are not auto-paired) ---
Type `name := "Ava"`
Sleep 1.2s
Ctrl+R
Sleep 1.5s        # signal lands — level passed

# --- next module ---
Ctrl+N
Sleep 3.5s        # Cold Readings — new mission, harder type

# --- solve level 2 (auto-paired () + overtype handle the parens) ---
Type `celsius := float64(raw) / 10`
Sleep 1.2s
Ctrl+R
Sleep 2.5s        # 21.5 — passed

# --- AI-generated mission (needs network + a wired Signal key) ---
Escape
Sleep 1s
Up                # Adventure -> Mission
Sleep 600ms
Enter             # open the topic popup
Sleep 800ms

# pick Slices: variables -> functions -> control-flow -> slices
Down
Down
Down
Sleep 800ms
Enter
Sleep 9s          # GOSCII generates and self-validates the mission (network)

# type a wrong stub — an unused slice triggers a compile error
Type `nums := []int{1, 2, 3}`
Sleep 1.2s
Ctrl+R
Sleep 2.5s        # compile fails — the error logs surface

# hand the logs to GOSCII's analyzer
Ctrl+G
Sleep 7s          # GOSCII reads the logs and explains (network)

Sleep 2s
