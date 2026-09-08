# SopeBox

A podcast player that lives in your terminal, built around one idea: **every
voice in the conversation gets its own orb.** As a podcast plays, SopeBox
listens for distinct speakers and adds a floating radial-spectrum orb for each
one. The orb of whoever is talking lights up and grows, the others drift and
dim, and the speakers' names sit underneath. Next to it, a captions pane
transcribes the show and animates an emoji whenever someone says a word that
deserves one (coffee, fire, rocket, laughing, …).

It borrows the folder-of-feeds browser from **castero**, the visualizer tuning
and online-art finder from **wvfrm**, and the animated emoji engine from
**G1yph**, then adds streaming or downloaded playback, Apple-Podcasts-style
automatic downloads, and transcripts from the feed or generated locally with
whisper.cpp.

```
 SopeBox   1 Now Playing   2 Podcasts   3 Queue   4 Downloads   5 Search   6 Settings   ctrl+k help
  1901 - Cryptic Warnings                              orbs · voices transcript ╭ captions · feed:srt ─────────╮
                 ⢀⣠⣴⣶⣦⣄⡀                                                        │        ▄▄▄▄▄▄▄▄▄▄              │
              ⣠⣾⣿⣿⣿⣿⣿⣿⣿⣷⣄            ⡀⢀⡀                                       │      ▄████████████▄            │
             ⣾⣿⣿⡿⠋⠉⠉⠙⢿⣿⣿⣷         ⣠⣾⣿⣿⣷⣄                                      │      ██▀▀▀▀▀▀▀▀▀▀██            │
            ⣸⣿⣿⡏   ⠈⢹⣿⣿⣇       ⢠⣿⣿⡿⠿⢿⣿⣿⡄                                     │       “coffee”  ☕             │
            ⣿⣿⣿⡇    ⢸⣿⣿⣿       ⢸⣿⣿⡇  ⢸⣿⣿⡇                                     │                                │
            ⢻⣿⣿⣧   ⢀⣼⣿⣿⡟       ⠘⣿⣿⣷⣤⣾⣿⣿⠃                                     │ Adam Curry: …and I said, before │
             ⠻⣿⣿⣿⣶⣶⣿⣿⣿⠟         ⠈⠻⣿⣿⣿⠟⠁                                      │ my second cup of coffee, that   │
               ⠉⠛⠿⠿⠛⠉             Rob Dew                                     │ the whole thing was a psyop.    │
                Adam Curry                                                     │ John: [laughs] Of course it was.│
  ● Adam Curry   ○ Rob Dew                sens 0.50 · smooth 0.70 · falloff 0.060 ╰────────────────────────────────╯
 14:02 ━━━━━━━━━━━━━━━━━━━┼━━━━━━━━━━━━━━━━━━━╸─────────────┼──────────────────────────── -2:41:10 / 2:55:12
 ▶ playing · 1.00x · vol 80% · downloaded · No Agenda                              queue 2 · theme sopebox · orbs
```

## Install

You need **Go 1.25 or newer** and **ffmpeg** (it decodes every audio format and
handles streaming and speed changes). `whisper.cpp` is optional: it generates
captions for podcasts whose feeds don't ship a transcript.

```sh
brew install go ffmpeg          # macOS; on Linux use your package manager
brew install whisper-cpp        # optional, for local transcription

git clone https://github.com/Cid-Emmerich/SopeBox
cd SopeBox
go build -o sopebox ./cmd/sopebox
sudo mv sopebox /usr/local/bin/ # or anywhere on your PATH
```

## First run

```sh
sopebox import castero          # bring in your castero subscriptions
sopebox import opml feeds.opml  # …or an OPML export from another app
sopebox search radiolab         # …or search iTunes and pick a show
sopebox add https://feeds.simplecast.com/EmVW7VGp   # …or paste a feed URL
sopebox                         # open the player
```

Podcast icons come from the feed automatically. If a show has none (or a bad
one), press **i** on it to search iTunes for its artwork, exactly like
`wvfrm`'s album-art finder.

## The voice visualizer

The now-playing screen is the point of the app. Eight styles, cycled with
**v**:

| style | what it does |
|---|---|
| `orbs` | one radial spectrum per voice, floating together with gentle physics; the active speaker drifts to the centre |
| `constellation` | voices as pulsing stars, joined by lines that light up along the flow of conversation (who answers whom) |
| `halo` | nested rings around a shared centre, one ring per voice |
| `ribbon` | a row of orbs above a scrolling who-spoke-when timeline |
| `wordflow` | orbs plus the words they say, drifting away from the speaker (loud words get a ✦) |
| `pulse` | a single breathing orb that changes colour with the speaker |
| `talktime` | a living donut of each person's share of the talking |
| `bars` | the classic analyser, tinted by whoever is speaking |

### How voices are told apart

SopeBox has two ways to know who is talking, chosen by the **voice mode**
(**M**, or in Settings):

- **transcript** — if the episode has a transcript with speaker labels
  (Podcasting 2.0 `<podcast:transcript>` JSON/VTT/SRT/HTML files, like No
  Agenda, Buzzsprout shows, Podnews…), orbs follow those labels exactly and
  are named after the speakers.
- **acoustic** — otherwise SopeBox listens. Each voiced stretch of audio is
  summarised as a fingerprint (spectral shape + pitch + brightness) and
  compared with the voices heard so far. A close match lights that orb; a
  voice that stays far from everyone for long enough earns a new orb. Names
  default to *Voice 1, Voice 2…*
- **auto** (default) uses the transcript when it has speakers, else listens.

Three knobs shape the acoustic isolation, mirroring wvfrm's
smoothing/sensitivity/falloff:

| key | setting | effect |
|---|---|---|
| `{` / `}` | **sensitivity** | lower merges similar voices into one orb; higher splits eagerly into more orbs |
| `(` / `)` | **smoothing** | how much each voice fingerprint is averaged over time |
| `<` / `>` | **falloff** | how quickly an orb dims after its speaker stops |

Voices are **remembered per podcast**: when you quit (or the episode changes)
the fingerprints of anyone who spoke for more than 15 seconds are saved with
the show, so the same hosts get the same orbs next episode. Press **N** to type
a name for the voice that is speaking, or **B** to pick one from the people the
feed declares (`<podcast:person>`). **K** merges the current voice into the
previous one when the tracker split one person in two; **X** clears this
episode's voices; **F** forgets everything learned for the show.

### Tuning the look

The wvfrm tuning keys are all here: **g/G** gradient, **i** orb fill (braille,
dots, rings, petals, ascii, block), **x** peak dots, **p** orb physics, **w/W**
rotation, **e/E** orb size, **,/.** smoothing, **[/]** gain, **;/'** falloff,
**L** speaker names, **R** reset. **t/T** cycle fourteen colour themes; the
`match` theme takes its colours from the podcast's icon. Every orb has its own
colour from the theme's voice palette.

## Captions and animated emoji

The captions pane (right side by default; **C** moves it to the bottom, **u/U**
resize it, **c** hides it) shows the current line with the spoken word lit,
recent lines above, the next line below, and chapter markers from
`<podcast:chapters>` (they also appear as ticks on the progress bar).

Whenever a spoken word matches one of ~550 trigger words, the corresponding
one of G1yph's 100 hand-drawn animated glyphs plays at the top of the pane:
*coffee*, *fire*, *rocket*, *laughing*, *love*, *cat*, *idea*, *rain*, *America*,
*billion*, and so on. Sound cues from transcripts work too (`[laughter]`,
`[music]`, `[applause]`), and a question gets a raised hand. **j** toggles
the emoji, **J** picks the render style (blocks, braille, ascii, chunky) and
**k** the colouring.

### Where transcripts come from

1. The feed. Podcasting 2.0 transcripts are fetched and cached automatically.
2. **whisper.cpp**, locally. Download the episode (**d**), then press **T**.
   The first run fetches the model (`base.en` by default, change it in
   Settings) into `~/.cache/sopebox/models`. Captions appear when it finishes;
   you can keep listening meanwhile. Set *auto transcribe* to `downloaded` in
   Settings to do this for every downloaded episode without a feed transcript,
   or run `sopebox transcribe <episode words>` from the shell.

Whisper output has word timestamps but no speaker labels; the acoustic
tracker still separates the voices. The `small.en-tdrz` model adds speaker
*turn* markers, which the captions use to break lines.

## Podcasts, downloads and the queue

The **Podcasts** view is castero's layout: shows on the left (with a tiny
icon each), episodes on the right, details underneath with the show's
artwork. **l** cycles the episode list between *this podcast*, *all recent
episodes* and *downloaded*. **/** filters by words. **enter** plays
(streaming, or from disk if downloaded), **e** queues, **E** plays next,
**d** downloads, **D** deletes a download, **m** marks played.

**Downloads** run in the background (2 at a time by default) into
`~/Podcasts/SopeBox/<Show>/<date> - <title>.mp3`. Like Apple Podcasts you can
have SopeBox keep the **1, 3, 5 or 10 most recent episodes** of every show
downloaded automatically (Settings → *auto-download latest*), override it per
show with **A** in the podcast list, and optionally have old automatic
downloads removed (*keep only latest*). The policy runs after every feed
refresh; press **a** in the Downloads view to apply it now.

Playback remembers where you were in every episode, resumes the last episode
on start, and continues with the queue when an episode ends. **s/S** change
speed with pitch preserved; **←/→** skip 10 s back / 30 s forward (shift for
4×).

## Command line

```sh
sopebox                          # open the player
sopebox add <feed url>           # subscribe
sopebox search <words>           # search iTunes, pick a number to subscribe
sopebox import castero           # import castero's subscriptions
sopebox import opml <file>       # import an OPML file
sopebox export <file.opml>       # export subscriptions
sopebox refresh                  # refresh every feed
sopebox list                     # list subscriptions
sopebox download [show] [n]      # download the n newest episodes (default 3)
sopebox transcribe <words>       # whisper.cpp on a downloaded episode
sopebox path <dir>               # set the download folder
sopebox -t synthwave -v halo     # start with a theme and a visualizer
```

## Files

```
~/.config/sopebox/sopeboxrc          settings (saved on quit; editable)
~/.local/share/sopebox/library.json  subscriptions, episodes, positions, learned voices
~/.cache/sopebox/icons               podcast artwork
~/.cache/sopebox/transcripts         cached captions
~/.cache/sopebox/models              whisper models
~/Podcasts/SopeBox                   downloads
```

Set `SOPEBOX_KITTY=0` to disable the Kitty graphics protocol (pixel-perfect
icons in Ghostty, Kitty, WezTerm and Konsole), or `SOPEBOX_KITTY=1` to force
it.

## Layout of the code

```
cmd/sopebox         command line entry point
internal/audio      ffmpeg-backed player: files and streams, seeking, speed
internal/dsp        FFT analyzer plus voice features (pitch, timbre, voicing)
internal/voices     the speaker tracker that turns features into orbs
internal/vis        the eight visualizer styles and orb physics
internal/captions   word → emoji triggers, the emoji animator, caption wrapping
internal/transcript feed transcripts (JSON/VTT/SRT/HTML), whisper.cpp, chapters
internal/feed       RSS with iTunes + Podcasting 2.0 namespaces, OPML, castero
internal/store      the library: podcasts, episodes, positions, queue, voice profiles
internal/download   the download queue and the auto-download policy
internal/art        icons: rendering, palette extraction, iTunes search, Kitty
internal/paint      G1yph's vector painter and rasterisers
internal/glyph      G1yph's 100 animated emoji
internal/theme      colour themes with per-voice palettes
internal/ui         the tcell interface
```

Run the tests with `go test ./...`.

## License

Released under the [MIT License](LICENSE). Copyright (c) 2026 Cid Emmerich.
