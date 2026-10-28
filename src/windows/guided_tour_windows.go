//go:build windows

package main

// The guided tour is a single, editable sequence. The exact Narration string
// is both the Field Manual's printed transcript and Piper's spoken input.
// IDs refer to existing controls, never to guessed screen rectangles.
type tourScene struct {
	Title     string
	Tab       string
	Page      string
	Focus     []int
	Narration string
	Demo      string
}

func scene(title, tab string, focus []int, narration string) tourScene {
	return tourScene{Title: title, Tab: tab, Focus: focus, Narration: narration}
}
func demoScene(title string, focus []int, narration, event string) tourScene {
	v := scene(title, "operations", focus, narration)
	v.Demo = event
	return v
}
func pageScene(title, page string, focus []int, narration string) tourScene {
	v := scene(title, "manual", focus, narration)
	v.Page = page
	return v
}

var guidedTour = []tourScene{
	// Audio FIRST. No spoken guide until the listener says they heard TEST VOICE.
	scene("01 — Hear Piper first", "settings", []int{ID_OUTPUT_COMBO},
		"First, choose the speakers or headset where you want to hear Piper. Use the Output Device box on the right. Then press Test Voice. If you hear the test, press the guide's Next button. If you hear nothing, choose a different output and test again. This first instruction is printed because you might not be able to hear me yet."),
	scene("02 — Welcome to RAWDOGS", "settings", []int{ID_TEST_VOICE},
		"Welcome to RAWDOGS Voice Controlled Field Calculator for the WARDOGS game. I am Piper, the voice that speaks to you. Vosk is a separate program that listens to you. In this guided tour I will explain every area, one control at a time. Use the blue and silver Walkman buttons above to play, pause, repeat, go back, or move forward. Nothing advances until you choose Next."),
	scene("03 — How the Walkman works", "settings", []int{ID_TOUR_PREV, ID_TOUR_PLAY, ID_TOUR_PAUSE, ID_TOUR_STOP, ID_TOUR_NEXT, ID_TOUR_RESTART},
		"These are the six Walkman controls at the top of every tab. Previous returns to an earlier topic. Play speaks or resumes the current topic. Pause holds your place. Stop silences this narration without deleting settings. Next advances to the following topic, and Restart takes you to the beginning. To finish the whole tour, open Settings and press the original End Guide button. I will open Settings and highlight that button when we reach the final lesson."),
	scene("04 — Audio output selection", "settings", []int{ID_OUTPUT_COMBO},
		"The Output Device menu chooses where Piper's voice is played. For most people, this will be their headphones or speakers. This is not your microphone. You can change the output if the voice is coming through the wrong device."),
	scene("05 — Test Voice", "settings", []int{ID_TEST_VOICE},
		"Test Voice plays a short Piper example through the selected output. If it is silent, check the Windows device, the output menu, the Voice Volume setting, and the Mute button before going further."),
	scene("06 — Speech speed", "settings", []int{ID_SPEED_COMBO},
		"Speech Speed controls how quickly Piper speaks. A slower rate may be easier to understand during setup. A faster rate makes firing-solution acknowledgements shorter during play. Try a speed that is comfortable for you; you can change it again later."),
	scene("07 — Voice volume", "settings", []int{ID_VOLUME_COMBO},
		"Voice Volume adjusts Piper's loudness separately from the Windows master volume. If speech sounds quiet, check both. A high volume can leak through a microphone and confuse recognition, especially when using speakers."),
	scene("08 — Mute", "settings", []int{ID_PIPER_MUTE},
		"Mute turns Piper's spoken output off or back on. The calculator can still show firing solutions visually. The guide cannot speak while Piper is muted, so use the written text if you need to restore your audio settings."),
	scene("09 — Stop Speech", "settings", []int{ID_PIPER_STOP},
		"Stop Speech interrupts Piper's current spoken message. It does not stop Vosk listening and does not erase coordinates. This differs from the guide's Stop control, which stops this instructional narration."),
	scene("10 — Microphone permission", "settings", []int{ID_PERMISSION_RUN},
		"This Run gives RAWDOGS permission to use your microphone for this session. It is useful if you only want temporary microphone access. Voice recognition cannot receive your spoken commands without permission."),
	scene("11 — Always allow", "settings", []int{ID_PERMISSION_ALWAYS},
		"Always Allow remembers microphone consent between sessions. Use this if you expect to use voice control regularly. The Not Now button postpones the voice feature; keyboard and mouse entry remain possible."),
	scene("12 — Windows microphone privacy", "settings", []int{ID_OPEN_PRIVACY},
		"Windows Mic Privacy opens the Windows microphone permission settings. If RAWDOGS sees no microphone sound, check that Windows allows desktop applications to use your microphone."),
	scene("13 — Select a microphone", "settings", []int{ID_MIC_COMBO},
		"The Microphone Input Device menu selects the microphone that Vosk will listen to. Your headphones are not automatically your microphone. Check that the correct input is selected if you have a headset, webcam, or multiple audio devices."),
	scene("14 — Test your microphone", "settings", []int{ID_TEST_MIC},
		"Test Mic starts a live microphone check. Speak normally and watch the level indicator. A moving level tells you RAWDOGS is receiving audio; it does not by itself mean Vosk has understood a command. Stop the test when you are finished."),
	scene("15 — Microphone input level", "settings", []int{3001},
		"The Mic Level indicator shows how much sound RAWDOGS is receiving after any selected gain. If the level stays at zero while you speak, check the input device and Windows privacy settings. If the level is constantly high, reduce the gain."),
	scene("16 — Microphone gain", "settings", []int{ID_GAIN_COMBO},
		"Input Gain adds amplification to a quiet microphone. Begin at zero decibels and increase only if your voice is genuinely too quiet. Too much boost amplifies background noise and can make Vosk less accurate."),
	scene("17 — Rescan audio", "settings", []int{ID_RESCAN_AUDIO},
		"Rescan Windows Audio refreshes the available microphones and output devices. Use this after plugging in a new USB headset or when Windows changes a sound device. It does not reset your calculator targets."),
	scene("18 — Understanding Vosk", "settings", []int{ID_VOICE_COMBO},
		"Vosk listens to your microphone and converts sounds into words. RAWDOGS then uses its current six-step state to decide whether those words are a useful command. Vosk may hear a word correctly but the command can still be inappropriate for the step you are on."),
	scene("19 — Recognition language", "settings", []int{ID_LANG_COMBO},
		"Recognition Language identifies the language and region of the listening model. The current six-step command system uses English words such as X-ray, Yankee, and New Target. Choosing another recognition language alone does not translate these commands."),
	scene("20 — Confidence threshold", "settings", []int{ID_CONF_COMBO},
		"Confidence Threshold controls how confident RAWDOGS expects a recognition result to be before accepting it. Higher values may reject more uncertain speech; lower values may accept more mistakes. Start with the default and change it only after checking the raw transcript."),
	scene("21 — Installed Vosk model", "settings", []int{ID_VOICE_COMBO},
		"Active Recognition Model shows which locally installed Vosk model is being used. A model must be installed and active before the listener can recognize speech. The model works offline after it has been downloaded."),
	scene("22 — Refresh installed models", "settings", []int{ID_REFRESH_VOICE},
		"Refresh Installed Models asks RAWDOGS to check the models already present on your computer. Use it if you have just installed a listening pack and it is not yet listed."),
	scene("23 — Vosk model library", "settings", []int{ID_VOSK_LIBRARY_COMBO},
		"The Vosk Model Library selector lists the available local packs to manage. Selecting an entry here is not the same as activating it. RAWDOGS keeps installation, active selection, and saved configuration separate so you can inspect alternatives without disrupting your setup."),
	scene("24 — Install or repair Vosk", "settings", []int{ID_VOSK_INSTALL_SELECTED},
		"Install or Repair checks and prepares the selected Vosk model. A model already installed should not need downloading again. Wait for the installation status before trying to select the model."),
	scene("25 — Use selected Vosk model", "settings", []int{ID_VOSK_USE_SELECTED},
		"Use Selected activates the installed listening model you chose. To keep that choice when you restart RAWDOGS, you must also press Save Setup to Disk. Do not confuse installed with active, or active with saved."),
	scene("26 — Piper voice library", "settings", []int{ID_ACTUAL_LIBRARY_COMBO},
		"Piper is the speaker, not the listener. The Piper voice library lists the speaking voices already available on this computer. You may keep several voices installed while only one is active at a time."),
	scene("27 — Install or repair Piper", "settings", []int{ID_ACTUAL_INSTALL_SELECTED},
		"Install or Repair prepares the selected Piper voice. Downloading may take time. A complete installed voice is then available for selection, but installing it does not automatically make it the active voice."),
	scene("28 — Use selected Piper voice", "settings", []int{ID_ACTUAL_USE_SELECTED},
		"Use Selected activates the chosen installed Piper voice. Then test it with Test Voice. Save Setup to Disk afterwards if you want RAWDOGS to remember the selection on its next launch."),
	scene("29 — Light and Dark themes", "settings", []int{ID_THEME_COMBO},
		"Theme offers Dark panels or Light panels. Both use the same controls, calculations and command colors. Dark is the established default. Light uses lighter panels with dark writing. Select the one you find easiest to read."),
	scene("30 — Team camouflage", "settings", []int{ID_TEAM_COMBO},
		"Team changes the repeating camouflage and team identification. Manticore is green, Lonestar is blue, and Valkyra is red. Team color is only a visual theme: choosing a different camouflage does not change your firing calculations or coordinate values."),
	scene("31 — Save the setup", "settings", []int{ID_SAVE_SETUP},
		"Save Setup to Disk writes your selected microphone, speakers, listening model, Piper voice, speed, volume and appearance settings to this computer. Selecting a choice is not always the same as saving it. After changing settings that you want to keep, use this green button."),
	scene("32 — Start and stop listening", "settings", []int{ID_SETTINGS_LISTEN},
		"Start Listening turns the Vosk recognizer on. Stop Listening turns it off. These controls appear in Settings, Operations, and Voice Training, but they all control the same listener. A stopped listener will not respond to spoken coordinates."),
	scene("33 — What the status monkeys mean", "settings", []int{TOUR_VOSK_MONKEY, TOUR_PIPER_MONKEY},
		"The Vosk monkey shows whether the listener is stopped, starting, or ready. The Piper monkey separately shows speaking activity. Their small region labels identify the chosen language or voice. One system can be ready even while the other is idle."),
	scene("34 — RAWDOGS command colors", "training", []int{ID_ADAPT_LEGEND},
		"RAWDOGS uses colors to tell you what is happening. Green means accepted or confirmed. Magenta identifies an active selection, an instructional outline, or information still being learned. Amber asks for attention or review, and red marks rejection or an error. On a coordinate edit, a blue outline identifies the keyboard field. Always check the text as well as the color."),
	scene("35 — Operations overview", "operations", []int{5030},
		"Operations is the working calculator. Actual means the position of your mortar. Target means where you want the shot to land. The current voice context tells you which command RAWDOGS is expecting, and the firing-solution panel shows the result when both positions are valid."),
	demoScene("36 — Safe scripted demonstration", []int{5001, 5002},
		"We will now watch a protected training demonstration. These examples are generated inside RAWDOGS, not heard through the microphone. They use the real coordinate acceptance and calculation functions, but are not a test of Vosk recognition. Your previous actual and saved targets are kept aside and restored when you finish the guide.", "begin"),
	demoScene("37 — Step one: Refresh Actual", []int{5001, 5002},
		"Step one is to announce your mortar position. Say Refresh Actual. The program then expects an X-ray coordinate. This activation cue prevents an ordinary number from being confused with a position. Watch as I demonstrate the same internal state change.", "actual_start"),
	demoScene("38 — Step two: Actual X-ray", []int{5001},
		"Step two is X-ray followed by the mortar's X coordinate. In this example it is one hundred point four two. Once X is accepted, RAWDOGS turns the confirmed value green and expects Yankee next.", "actual_x"),
	demoScene("39 — Step three: Actual Yankee", []int{5002},
		"Step three is Yankee followed by the mortar's Y coordinate. Here it is one hundred and fourteen point five five. When Yankee is accepted, RAWDOGS confirms the whole Actual automatically. You do not need an extra Over command in the six-step flow.", "actual_y"),
	demoScene("40 — Step four: New Target", []int{ID_OP_NEW_TARGET},
		"Step four is New Target, or Refresh Target. This tells RAWDOGS that the next coordinates describe a target rather than the mortar. That distinction is why the six-step logic helps prevent misunderstandings.", "target_start"),
	demoScene("41 — Step five: Target X-ray", []int{5011},
		"Step five is X-ray followed by the target's X coordinate. Our training target is one hundred and three point four two. RAWDOGS accepts this X value and moves on to the target Yankee step.", "target_x"),
	demoScene("42 — Step six: Target Yankee", []int{5012},
		"Step six is Yankee followed by the target's Y coordinate, one hundred and eighteen point five five. RAWDOGS now confirms the target and calculates a firing solution using the real L81 calibration. The example distance is five hundred metres, within its calibrated range.", "target_y"),
	scene("43 — Understanding bearing, range and MIL", "operations", []int{5032},
		"The firing solution shows bearing, range in metres, and the mortar elevation in MIL. The values come from the actual and target coordinates, not from Piper's narration. If a target is outside the photographed L81 calibration range, RAWDOGS will not invent a MIL value."),
	scene("44 — Why the voice logic matters", "operations", []int{5030},
		"Notice how RAWDOGS always knows which piece of information is required. If it is waiting for Yankee, saying another X-ray does not fill the missing Y field. Check the current voice context and use the cue it asks for. A rejected cue is not automatically a broken microphone."),
	scene("45 — Current command and transcript", "operations", []int{5031},
		"The Operations Transcript shows what the calculator has accepted and its acknowledgements. If something seems wrong, check this record before repeating the command. It helps separate a recognition problem from a coordinate that has not yet been confirmed."),
	scene("46 — Raw speech, confidence, interpretation", "operations", []int{5052},
		"The live recognition area shows Vosk's raw words, confidence and RAWDOGS interpretation. Raw heard is what the recognizer wrote. Interpretation is what the command logic understood. This difference becomes particularly useful when teaching a personal pronunciation."),
	scene("47 — Saved target memory", "operations", []int{5020, 5021, 5022, 5023, 5024},
		"Target Memory contains four saved target slots and an additional unsaved working slot. Click a stored target to restore its firing solution. Creating another target does not automatically require changing the mortar's Actual position."),
	demoScene("48 — Recall a saved target", []int{5020},
		"Recall Target selects a previously confirmed target and shows its firing solution again. The demonstration now recalls the first saved target. A target that has not been stored cannot be recalled.", "recall"),
	scene("49 — Repeat the solution", "operations", []int{5032},
		"Repeat Solution asks Piper to say the current bearing, range and MIL again. There are also commands for repeating bearing, range or MIL separately. The display remains available even if Piper is muted."),
	demoScene("50 — Clear one target", []int{ID_CLEAR_TARGET_1},
		"Clear Target removes only the selected saved target. Here I am clearing target one in our temporary demonstration. The Actual position is not removed. During normal operation, Undo Clear can restore your most recent deletion.", "clear_target"),
	demoScene("51 — Moving the mortar: Refresh Actual", []int{5001, 5002},
		"If you move to a different mortar nest, the old firing position is no longer correct. Say Refresh Actual, then enter the new X-ray and Yankee coordinates. The previous confirmed Actual is protected until both replacement values have been accepted. Our demonstration now establishes a new position.", "move_actual"),
	demoScene("52 — Create a new target after moving", []int{ID_OP_NEW_TARGET},
		"Now that our mortar is at its new nest, we can create a fresh target without restarting RAWDOGS. Say New Target. The calculator again expects X-ray followed by Yankee. This is the correct sequence whenever you move and then designate a new target.", "new_target_start"),
	demoScene("53 — New target X-ray", []int{5011},
		"Our second example target has an X-ray coordinate of one hundred and four point four two. Watch the target X field accept it. The confirmed mortar position remains untouched.", "new_target_x"),
	demoScene("54 — New target Yankee and solution", []int{5012, 5032},
		"The new target Yankee is one hundred and eighteen point five five. RAWDOGS accepts the second coordinate, confirms the target, and calculates another real in-range firing solution from the moved mortar position.", "new_target_y"),
	demoScene("55 — Demonstrate Correction", []int{5012},
		"If I gave the wrong Yankee value, I can say Correction and enter its replacement. Here I correct the most recent target Yankee to one hundred and nineteen point five five. The other coordinate stays in place, and RAWDOGS recalculates the real result.", "correct_y"),
	demoScene("56 — Demonstrate Cancel", []int{ID_OP_CANCEL},
		"Cancel is for an unfinished entry. In this example RAWDOGS begins another new target, then cancels it before any coordinates are confirmed. Our already confirmed Actual and target stay available. Cancel is not the same as Clear All Targets.", "cancel_draft"),
	scene("57 — Correct, cancel, restart", "operations", []int{ID_OP_CANCEL},
		"Correction reopens the most recent coordinate so you can replace it. Cancel abandons an unfinished entry but keeps already confirmed positions. Restart or Start Again resets the unfinished target chain. Backspace can move backward through keyboard entry, while Tab and Enter move through expected fields."),
	scene("58 — Clear Actual and clear all targets", "operations", []int{ID_OP_CLEAR_ALL, ID_OP_CLEAR_ACTUAL},
		"Clear All Targets removes the saved and unsaved target information but preserves Actual. Clear Actual is a separate action for when the mortar position should be removed. Undo Clear can restore the most recent supported deletion. Check the button label carefully before pressing it."),
	scene("59 — New target, Confirm and Cancel", "operations", []int{ID_OP_NEW_TARGET, ID_OP_CONFIRM, ID_OP_CANCEL},
		"New Target begins a target entry using mouse or keyboard. Confirm accepts a complete pending entry where confirmation is appropriate. Cancel discards the current provisional entry. These buttons are different from deleting a previously saved target."),
	scene("60 — Voice Training: why it exists", "training", []int{ID_ADAPT_EXPECTED},
		"Voice Training helps RAWDOGS understand how a particular operator speaks. Vosk sometimes writes free when a person meant three, or to when they meant two. These words can sound alike. RAWDOGS uses both the current command step and confirmed personal pronunciation corrections rather than arguing over spelling."),
	scene("61 — Personal operator name", "training", []int{ID_ADAPT_OPERATOR},
		"The Operator Profile name identifies whose pronunciation corrections are being used. Default is the initial profile name, not the selected Vosk model. Enter your own name if different people share this computer."),
	scene("62 — Load or create a profile", "training", []int{ID_ADAPT_LOAD_PROFILE},
		"Load or Create Profile looks for the operator name you typed. If the profile exists, it loads the saved corrections. If it does not, RAWDOGS creates a new local profile. It does not automatically upload the data or replace another operator's confirmed rules."),
	scene("63 — Which Vosk model is being trained", "training", []int{ID_ADAPT_MODEL},
		"This label shows the Vosk model associated with the operator profile. Choose your active model in Settings, not here. Changing the listener model can change which pronunciation corrections are relevant."),
	scene("64 — Start Listening for training", "training", []int{ID_TRAIN_LISTEN},
		"Press Start Listening so Vosk can hear your practice words. The same listener is shared with Operations. The microphone must be selected, permitted and working before you start a calibration exercise."),
	scene("65 — Start Calibration", "training", []int{ID_ADAPT_START},
		"Start Calibration begins the voice-training exercises. Piper will request a word or phrase on the screen. Read that exact phrase aloud. RAWDOGS compares what Vosk heard with the expected phrase, but it does not learn anything permanently without your confirmation."),
	scene("66 — Piper's training request", "training", []int{ID_ADAPT_EXPECTED},
		"This large training line shows exactly what Piper wants you to say. Read the words shown here, not the previous result. If you accidentally say the wrong phrase, repeat the exercise rather than teaching RAWDOGS an incorrect correction."),
	scene("67 — Vosk Heard and RAWDOGS Interprets", "training", []int{ID_ADAPT_HEARD, ID_ADAPT_INTERPRETATION},
		"Vosk Heard displays the words the recognition engine actually returned. RAWDOGS Interprets shows the meaning inferred after the current command context and any active corrections. If Vosk writes free but the command expects a number, the intended interpretation may be three."),
	scene("68 — Confirm Training", "training", []int{ID_ADAPT_CONFIRM},
		"Confirm Training accepts the proposed correction only when you know the requested phrase and your spoken answer matched your intention. RAWDOGS may require repeated confirmations before a personal rule becomes active. Do not confirm a mistake merely to move to the next exercise."),
	scene("69 — Repeat Exercise", "training", []int{ID_ADAPT_REPEAT},
		"Repeat Exercise asks for the current practice phrase again. Use it when you misheard Piper, spoke the wrong words or need another attempt. Repeating is better than confirming an uncertain training result."),
	scene("70 — Skip Exercise", "training", []int{ID_ADAPT_SKIP},
		"Skip Exercise moves past the current phrase without accepting it as a training correction. Use this if an example is not useful, or you would rather practise something else. Skipping does not mean a pronunciation has been learned."),
	scene("71 — Stop Calibration", "training", []int{ID_ADAPT_STOP},
		"Stop Calibration ends the training exercise sequence. It does not erase active personal pronunciation rules. Stop Listening is a different button: that one turns off the shared microphone recognizer."),
	scene("72 — Clear Transcript", "training", []int{ID_TRAIN_CLEAR},
		"Clear Transcript clears the displayed speech history so you can start reading a fresh set of results. It does not remove confirmed pronunciation rules from your operator profile."),
	scene("73 — Cue Reference", "training", []int{ID_ADAPT_REFERENCE},
		"Cue Reference switches the lower-right list between your personal pronunciation rules and the original RAWDOGS command vocabulary. The original cue list is a reference, not your trained voice profile."),
	scene("74 — Personal pronunciations and colors", "training", []int{ID_ADAPT_RULE_LIST},
		"The Personal Pronunciations list shows words Vosk heard, what RAWDOGS should interpret, and the rule status. Green is an accepted active correction. Amber is still waiting for review. Magenta shows a learning stage. Red means rejected. Your existing examples include free for three, for for four, and to for two."),
	scene("75 — Remove Selected", "training", []int{ID_ADAPT_REMOVE},
		"Remove Selected deletes the pronunciation rule you have selected in the personal rule list. It does not delete the whole profile, nor does it remove words from Vosk itself. Choose the rule carefully before using this button."),
	pageScene("76 — Field Manual overview", "quick", []int{ID_MANUAL_QUICK},
		"The Field Manual has six pages with different jobs. Quick Radio Flow provides the shortest six-step instructions. Voice Cues contains corrections and special spoken commands. You can use these pages as a printed reference while practising."),
	pageScene("77 — Voice Cues page", "cues", []int{ID_MANUAL_CUES},
		"The Voice Cues page is your reference for X-ray, Yankee, corrections, Cancel, Restart, recall, clearing targets and spoken readbacks. When a voice command does not work, check the expected current step before changing your pronunciation."),
	pageScene("78 — Actual's Voice page", "voices", []int{ID_MANUAL_VOICES},
		"Actual's Voice explains Piper output, choosing voices, speaker volume and speed. Remember that Piper is the speaker. Vosk is the listener. You can use the calculator without spoken output by reading the displayed results."),
	pageScene("79 — Vosk and Piper explained", "models", []int{ID_MANUAL_MODELS},
		"The Vosk and Piper page explains the two offline speech systems. They work independently. The status monkeys show which system is ready and which is stopped or speaking. Using headphones helps prevent Piper's words from being picked up as microphone input."),
	pageScene("80 — International Libraries", "libraries", []int{ID_LIBRARY_COUNTRY},
		"Libraries is the catalog browser. Choose a country, region or language to explore published listening models and speaking voices. The flags help you browse; they do not mean that every model understands RAWDOGS's English command vocabulary."),
	pageScene("81 — Choose Vosk or Piper", "libraries", []int{ID_LIBRARY_VOSK_MODE, ID_LIBRARY_PIPER_MODE},
		"Choose Vosk Listening when you want a speech-recognition model, or Piper Speaking when you want a voice that reads the results. The two catalogs serve different purposes. A speaking voice does not improve microphone recognition."),
	pageScene("82 — Select a library entry", "libraries", []int{ID_LIBRARY_CHOICES},
		"The list contains available library entries. The magenta row shows what you have selected for inspection, not necessarily what is active. An installed pack is available on your computer; the active pack is the one selected for use in Settings."),
	pageScene("83 — Preview a Piper voice", "libraries", []int{ID_LIBRARY_PREVIEW},
		"Preview Voice lets you hear a published voice sample before installing a Piper voice when a sample is available. Previewing a voice is not an installation and it does not change your active Piper voice."),
	pageScene("84 — Install or remove a pack", "libraries", []int{ID_LIBRARY_INSTALL, ID_LIBRARY_REMOVE},
		"Install Selected downloads and verifies the chosen pack. Watch the progress and completion message. Remove Installed removes a local pack only when it is safe to do so; an active or saved pack must first be replaced. A failed download should leave the working model untouched."),
	pageScene("85 — Return to Settings and save", "libraries", []int{ID_LIBRARY_SETTINGS},
		"Open Settings, Use Selected, and Save Setup to Disk are separate actions. After installing a pack here, return to Settings, select it, test it, and save your configuration. Do not assume an installation changes the active voice automatically."),
	pageScene("86 — The written and spoken guide", "narrated", []int{6206},
		"Page four, Narrated Guide, contains the indexed text of every section you have heard. The spoken words come from that same text, so changing one section's narration in a future update also changes its displayed wording. You can scroll and read the guide at any time."),
	scene("87 — Finish and operate normally", "settings", []int{ID_GUIDE_END},
		"You have reached the end of the guided tour. The End Guide button is here in Settings. Press it to remove the training outlines and restore your normal calculator workspace. Your example coordinates were a temporary demonstration. In live use, start listening and enter your own Actual and Target. You can return to Settings and start this tour again whenever you wish."),
}

func guidedTourNarration() []string {
	out := make([]string, len(guidedTour))
	for i, s := range guidedTour {
		out[i] = s.Narration
	}
	return out
}

func guidedTourIndex() string {
	var out string
	for i, s := range guidedTour {
		if i > 0 {
			out += "\r\n\r\n"
		}
		out += s.Title + "\r\n" + s.Narration
	}
	return out
}
