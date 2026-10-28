# RAWDOGS V.C.F.C. — v0.9.4 Preview Test Checklist

**Do not overwrite your working versions.** Extract this preview into its own folder. Windows UI, Piper audio and the Buy Me a Coffee click must be tested on your computer; the automated tests cannot prove those behaviors.

## Header and visibility

1. Launch in a **nonmaximized window**, then maximize and resize. The title block must display only **RAWDOGS** and **V.C.F.C.**; the full name and preview version appear separately in the Windows title bar.
2. Confirm exactly six clean blue-and-silver Walkman buttons at the top: Previous, Play, Pause, Stop, Next, Restart. No black artifacts, stray lettering, GUIDE READY label, or duplicate END button. The same buttons are available on each of the four tabs.
3. Confirm the **BUY ME A COFFEE** button is compact and clickable across its entire surface and opens `https://buymeacoffee.com/silvesta` in your default browser. The application stays open.
4. Confirm no overlap with the title, Walkman buttons, donation button, team banner, or lower navigation at narrow and wide window sizes. Check Light/Dark and all three camouflage teams if convenient.

## Narrated guide and original End Guide control

5. On Settings, press **START GUIDE**. The opening written instruction asks you to select output, press **TEST VOICE**, and continue only after you have heard Piper. There must be no reliance on narration before the output test.
6. Go to chapter **03 — How the Walkman works**. Piper should explain all six top controls, then clearly say that the **original END GUIDE is in Settings**, not in the header. Check that the written text says the same.
7. Press Next, Previous, Play, Pause, Stop and Restart during the guide; verify Piper, the selected chapter, the appropriate tab and magenta control outline remain synchronized. Switching between tabs must not resurrect a removed header END button or leave stray spotlights.
8. Jump to chapter **87 — Finish and operate normally** using Field Manual → Narrated Guide page 4. The app must open **Settings** and place a **precise magenta outline** around the existing **END GUIDE** button in the Settings bottom row. Piper must refer to **that button**, never an END button above the Walkman.
9. Click **Settings → END GUIDE**. Narration stops, training magenta highlights disappear and the operator's previous Actual and Target state is restored.

## Calculator and Voice Training regression

10. Visit Operations and confirm existing coordinate fields, target slots, calculated bearing/range/MIL, live recognition and transcript remain unchanged. In guide chapters 36–56, the scripted demonstration should use the real calculator without modifying saved real coordinates.
11. Check Voice Training: operator profiles, pre-existing personal FREE→THREE, FOR→FOUR and TO→TWO corrections, and exercises remain. The scripted demonstration must not train the operator's real profile.
12. Check the Field Manual's six pages and page 4's complete indexed script. The exported Markdown in this ZIP is derived from the same master narration source; text and spoken instructions should match.

Report any difference with a screenshot, window size, selected tab, chapter number, and what Piper said. Preserve the older build until this preview passes Windows testing.
