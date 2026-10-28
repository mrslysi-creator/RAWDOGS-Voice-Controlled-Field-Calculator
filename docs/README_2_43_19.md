# RAWDOGS Voice Controlled Field Calculator — Preview build 0.9.4 (clean Walkman, single End Guide)

**Windows native test build; not a public release.**

This ZIP is a *new experimental copy* based on the tested 2.43.18 UI collision hotfix. It does not overwrite your older 2.43.18 ZIP, archived 2.43.12 checkpoint, or existing public GitHub project. The application continues to use the established local user-profile and settings locations.

## Brand, guide controls, and optional support link

The window identifies itself as **RAWDOGS Voice Controlled Field Calculator**. The header layout follows the owner's annotated screenshot (which is a design specification, **not** a screenshot of this revised executable): **orange** = compact two-line title, **green** = six enlarged blue/silver Walkman icons, **orange coffee outline** = compact clickable support button. The existing team identification banner stays at far right; the four sections do not overlap, including at narrow window sizes. All six Walkman controls remain synchronized across tabs. The Settings END GUIDE button is the only explicit end-guide button.

The compact **BUY ME A COFFEE** button, placed in its own header area to the right of the Walkman controls, opens the existing project support page in the default browser: https://buymeacoffee.com/silvesta . This is entirely optional; no donations are required. The external link is opened only when clicked and does not affect the calculator or guide.

## Header layout history and current appearance

- **Removed** the extra `GUIDE: READY` caption/control completely. Guide progress remains visible in the indexed Field Manual text and existing status messages; no replacement header panel was added.
- **Enlarged** the original guide buttons from 46 to 70 pixels tall in the previous preview, and in this build replaced the low-resolution icon images with clean high-resolution blue/silver artwork.
- **Resized** the donation control from a long gold strip into the compact orange-marked button; its whole surface remains clickable and points to https://buymeacoffee.com/silvesta .
- **Kept the title compact** in its marked orange area. The existing team banner and the four main tabs are untouched.
- The calculator mathematics, six-step voice state machine, target memory, pronunciation profiles and model management are preserved. The guide narration was updated only where it previously referred to the deleted top END button.

## Preview build 0.9.4 — Compact title and clear controls

The compact header now reads **RAWDOGS** on its first line and **V.C.F.C.** on its second line, with **no version number in the header name**. The full official application name remains **RAWDOGS Voice Controlled Field Calculator** in the Windows title bar. The preview build number is shown separately there for test identification. The header title size and donation button remain compact. All six Walkman buttons use refreshed high-resolution blue-and-silver icons with clean symbols and no unwanted lines. The duplicate top END button has been removed completely; the original Settings → END GUIDE button is the only tour-finishing control. The complete spoken and written tour now consistently directs operators there, and lesson 87 opens Settings and highlights it.

## What is new

**87 carefully indexed Piper lessons.** User-controlled START GUIDE → END GUIDE spans **Settings, Operations, Voice Training and all six Field Manual pages**. Written text and Piper WAV are generated from the same master sequence (`guided_tour_windows.go`). Reading copy: `GUIDE_INDEX_AND_SCRIPT_2_43_19.md`.

**Sound output first.** The app starts on Settings. Press START GUIDE, choose headset/speakers, press TEST VOICE and only then click NEXT if you actually heard Piper. The first instruction is on-screen text because audio may not be available. The rest of the guide explains the full interface including microphone permissions/gain, Vosk listening, Piper speed/volume/voices, installation status, themes/camouflage, command colors, all six coordinate stages, target memory, voice training, and recovery steps.

**One shared Walkman.** Six refreshed blue/silver icon controls (Previous, Play, Pause, Stop, Next, Restart) live in the unused upper header across all tabs. The existing Field Manual audio buttons mirror that same tour controller while the tour is active. Guide chapters do not auto-advance merely because Piper finishes speaking. PAUSE/STOP holds the current section; Next/Previous can be used to review sections without losing your place.

**Real calculator demonstration, simulated operator.** During the Operations lessons, preprogrammed ACTUAL / TARGET input events go through the existing six-step coordinate processor and use the *real L81 maths*. No microphone is involved in the scripted example. The sample starts at Actual X100.42 Y114.55 and Target X103.42 Y118.55 (500 m). It demonstrates target storage, recall, clearing, moving to a different mortar position, a fresh target, correction, and cancellation. Your live target/actual workspace is snapshotted and restored at END GUIDE. The listener is deliberately stopped during a guide, and demonstration events are excluded from the voice diagnostic CSV. The real operator pronunciation rules are not trained or edited by the demonstration.

**Precision magenta outlines.** Outlines track actual Win32 child control rectangles, using the *closed* combobox size rather than its invisible dropdown. The Vosk/Piper status monkeys have two small explicit spotlight boxes. A different guide chapter removes the old focus rather than leaving ghost overlays. The light theme has a small charcoal backing around the monkey pictures; the dark theme is not changed.

## Installation (testers)

1. Extract the entire ZIP to a fresh folder. Do not extract it over a 2.43.18 folder.
2. Run `RAWDOGS_Voice_Controlled_Field_Calculator_v0.9.4_Preview.exe`. The supplied Piper/Vosk voice packs and your personal settings are **not** bundled into the ZIP. The app is designed to reuse existing local installed packs; a fresh machine may need to download default offline models and use the built-in setup.
3. Start on **Settings**, choose your output device, click **START GUIDE**, then **TEST VOICE**. Press **NEXT** only if you could hear the test.
4. Watch the small magenta outlines and click the upper Walkman controls. Visit page 4 of the Field Manual to jump through the entire indexed transcript.
5. When the guided demonstration finishes, open **Settings → END GUIDE** and verify that your real Actual/Target state has been restored.

**Known validation limit:** Go unit tests and cross-compilation succeeded on Linux, but Windows UI geometry, actual Piper audio/device selection, real downloads, and synchronization have **not yet been tested on Windows**. Treat this release as an interactive prototype for the owner, not a proven public beta.

**Scope control:** The established six-stage recognition logic, personal profile file handling, weapon calibration, calculation engine, field-manual reference pages (except new page-four tour), targets, camera textures, and icon are not intentionally modified. Review the results in Windows before distributing.

## Source and checks

The ZIP contains the complete Go source (`*.go`, `*.syso`) and original imagery (`assets/`, `camo_*.png`, `RAWDOGS_2_42_TEST.ico`). `guided_tour_windows.go` is the master script. `guided_tour_runtime_windows.go` is the synchronized controller. `guided_demo_test.go` verifies real L81 range math on the sample targets. Other calculation and parser tests remain included.

For developers with Go on Windows: `GO111MODULE=off go test ./...` and `GO111MODULE=off go build -o RAWDOGS.exe .`; run from the unpacked source folder. The supplied EXE is a native Windows 64-bit build.

The project is for the WARDOGS computer game. It is not a real-world artillery fire-control system.

## Guide ending: narration and highlight

Chapter **03** explicitly tells the operator where to find **Settings → END GUIDE**. Chapter **87** automatically selects Settings, displays the original **END GUIDE** control in a precise magenta outline, and has Piper explain that this is the button that ends the tour. The header contains **no END button**. The written guide copy matches the new spoken explanation word for word.
