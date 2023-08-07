
#settings.decoder.file_extensions.taglib.set([""])

## Tweak polly protocol to include -key (musical key of voice 0 = cmaj 12 = cmin amaj =4 24 = bmin) and other var (r = rate) or parsody

settings.init.allow_root.set(true)
settings.frame.audio.samplerate := 48000
settings.scheduler.fast_queues := 4
settings.scheduler.generic_queues := 4
settings.scheduler.non_blocking_queues := 4
settings.log.level := 3


settings.protocol.aws.polly.voice := "Aditi"
settings.protocol.aws.polly.format := "mp3"
interactive.harbor()

s1bpm = ref(1.0)
s2bpm = ref(1.0)

s1type = ref(false)
s2type = ref(false)
s3type = ref(false)
s4type = ref(false)

s2_verb = ref(false)

words = ref(false)

loud = ref("6")
notPlayAlone = ref(true)
spec_band_norm = ref("6")
ambient_only = ref(false)

music_centroid = ref("800.0")

music_now_playing = ref("silence by silence")
music_now_url = ref("null")

field_recording_now_playing = ref("silence by silence")
field_now_url = ref("null")

vocal_now_playing = ref("silence by silence")
vocal_now_url = ref("null")

music_push_q_length = ref(0)
field_push_q_length = ref(0)

music_playlog = playlog(duration=36000., persistency="./music_playlog", hash=fun (m) -> file.digest(metadata.filename(m)))
field_playlog = playlog(duration=36000., persistency="./field_playlog", hash=fun (m) -> file.digest(metadata.filename(m)))
vocal_playlog = playlog(duration=36000., persistency="./vocal_playlog", hash=fun (m) -> file.digest(metadata.filename(m)))
v_fx_playlog = playlog(duration=36000., persistency="./vfx_playlog", hash=fun (m) -> file.digest(metadata.filename(m)))


last_music_mix_type = ref("")

last_radio_metadata = ref([])

#### vocal trigger flags
v_fx_play = ref(false)
def play_v_fx()
  log("allowing effected vocal to play")
  v_fx_play := true
end
thread.when({10m or 40m},play_v_fx)

vox_play = ref(false)
def play_vox()
  log("allowing vocal to play")
  vox_play := true
#  ambient_only := true
end
thread.when({55m},play_vox)
#thread.when({55m or 05m or 15m or 25m or 35m or 45m},play_vox)

field_play = ref(false)
def play_f()
  log("allowing field to play on music stream")
  field_play := true
end
thread.when({36m},play_f)

poem_play = ref(false)
def play_p()
  poem_play := true
end
thread.when({25m}, play_p)

is_night = ref(false)

def is_night_calc()
    if time.local().hour > 19 or time.local().hour < 7 then
        is_night := true
    else
        is_night := false
    end
end

thread.when({0m}, is_night_calc)

is_night_calc() 

dusk_dawn = ref(false)

def dusk_dawn_calc()
    if time.local().hour == 20 or time.local().hour == 7 then
        dusk_dawn := true
    else
        dusk_dawn := false
    end
end

auth = ref(string.base64.encode("< username:password >"))

### next.py returns annotated url to next track
def nextm_alone() =
  print(time.local().hour)
  print(is_night())
  l = loud()
  sbn = spec_band_norm()
  nPA = if notPlayAlone() or is_night() then "Mix" else "" end
  ao = if ambient_only() or dusk_dawn() then "Ambient" else "" end
  uri = "http://localhost:8080/next?stream=music&loudness=#{l}&bandwidth_norm=#{sbn}&mix_type=#{nPA}&genre=#{ao}"

  result = http.get(headers=[("Authorization", "Basic " ^ auth() )], uri)

  request.create(result)
end

def nextb() =
  uri = "https://dev.rapidcarrot.com/next?stream=noise"
  result = http.get(headers=[("Authorization", "Basic " ^ auth() )], uri)
  request.create(result)
end

def nextm_b() =
  l = loud()
  uri = "https://dev.rapidcarrot.com/next?stream=noise&loudness=#{l}"
  result = http.get(headers=[("Authorization", "Basic " ^ auth() )], uri)
  request.create(result)
end

def nextv() =
  uri = "https://dev.rapidcarrot.com/next?stream=vocal"
  result = http.get(headers=[("Authorization", "Basic " ^ auth() )], uri)
  request.create(result)
end

def nextvfx() =
  uri = "https://dev.rapidcarrot.com/next?stream=vocal_fx"
  result = http.get(headers=[("Authorization", "Basic " ^ auth() )], uri)
  request.create(result)
end

def nexttxt() =
  text = music_now_playing()
  question = string.split(separator="by", text)[0]
  proc = "../python_scripts/get_poem.py #{question}"
  result = list.hd(default="", process.read.lines(proc))
  log("Writting a poem about #{question}")
  request.create(result)
end

s1 = request.dynamic(id="music_alone", prefetch=1, nextm_alone)
s1_q = request.dynamic(id="music_q", prefetch=0, nextm_b)

s2_q = request.dynamic(id="field", prefetch=0, nextb)
#s2_q_push = request.dynamic.list(id="field_q")

s3_q = request.dynamic(id="vocal", prefetch=0, nextv)

s4_q = request.dynamic(id="vocal_fx", prefetch=0, nextvfx)

poems = request.dynamic(id="poems", prefetch=0, nexttxt)

s1 = fallback([s1_q, s1])
#s2_fallback = fallback([s2_q_push, s2_q])
s2_fallback = s2_q

s1 = amplify(id="alone_amp", 1.0, s1)
s2 = amplify(id="field_amp", 0.22, s2_fallback)
s3 = amplify(id="vox_amp", 1.2, s3_q)
s4 = amplify(id="vox_fx_amp", 0.4, s4_q)

last_music_key = ref("")


## Trigger stream on startup  for debugging
#thread.run(fun() -> s3_q.set_queue(nextv()))
## map metadata
###
###
## set default value
def default(default_val, val) =
  if val == "" then default_val else val end
end
def map(m)
  type = m['stream']
  title = m['title']
  artist = m['artist']
  source_url = m['source_url']
  source_url_fallback = "https://duckduckgo.com?q="^title ^ "+" ^ artist

  ### change m['loud'] to m['norm_energy']
  loudness = default("5", m['norm_energy'])
  spec_band_norm_val = default("5", m['spec_bandwidth_norm'])
  log("playing - #{title} on #{m['source']}")
  
  log("s3 - #{s3type()}" )

  def increase_music_loud_val()
    loud := string(int_of_string(loud()) + 1)
    log("loud equals #{loud()}")
  end

  if type == "music" then
    if music_playlog.last(m) < 3600. then
      if s3type() then
          log("vocal")
      else
          log("repeat")
      end
      log("skipping music - either vocal is playing and music choice is not Amb or a repeat has been stopped")
#      thread.run(increase_music_loud_val)
      s1.skip()
      s1type := false
      []
    elsif m['mix_type'] == "Play alone" then
      if s3type() then
        log("skipping music - vocals are playing and play alone has been selected")
        s1.skip()
        s1type := false
        []
      else
        log("Words set to true")
        url = if source_url == "" then source_url_fallback else source_url end
        music_now_url := "#{url}"
        words := true
        s1type := true
        loud := loudness
        spec_band_norm := spec_band_norm_val
        notPlayAlone := true
#        music_centroid := default("800.0", m["dominant_frequency"])
        #music_now_playing := "#{title} by #{artist}"
#        last_music_key := m['key']
        m
      end
    else
      log("Words set to false")
      url = if source_url == "" then source_url_fallback else source_url end
      music_now_url := "#{url}"
      words := false
      s1type := true
      notPlayAlone := false
      loud := loudness
      spec_band_norm := spec_band_norm_val
#      music_centroid := default("800.0", m["dominant_frequency"])
      #music_now_playing := "#{title} by #{artist}"
#      last_music_key := m['key']
      m
    end
  elsif type == "noise" then
    if int_of_string(loudness) > 5 then
      s2_verb := true
    else
      s2_verb := false
    end

    if words() or (field_playlog.last(m) < 18000.) then
      log("skipping field rec - play alone playing or repeat")
      s2type := false
#      s2_q.set_queue([])
#      s2_q.skip()
      []
    else
      url = if source_url == "" then source_url_fallback else source_url end
      field_now_url := "#{url}"
      #field_recording_now_playing := "#{title} by #{artist}"
      s2type := true
      m
    end
  elsif type == "vocal" then
    if words() or (vocal_playlog.last(m) < 18000.) then
      log("skipping vocal - play alone playing or repeat")
#      s3_q.set_queue([])
      s3_q.skip()
      s3type := false
      []
    else
      url = if source_url == "" then source_url_fallback else source_url end
      vocal_now_url := "#{url}"
      #vocal_now_playing := "#{title} by #{artist}"
      #loud := "-1"
#      s3type := true
#      ambient_only := true
      m
    end
  elsif type == "vocal_fx" then
    if words() or s3type() or (v_fx_playlog.last(m) < 1800.) then
      log("skipping v fx")
 #     s4_q.set_queue([])
      s4_q.skip()
      s4type := false
      []
    else
      s4type := true
      m
    end
  else
    log("Something not expected has happened in Map metdata")
    m
  end
end

#s1_mix = metadata.map(update = false, strip = true, map, s1_mix)
s1 = metadata.map(update = false, strip = true, map, s1)
#s2_s1 = metadata.map(update = false, strip = true, map, s2_s1)
s2 = metadata.map(update = false, strip = true, map, s2)
s3 = metadata.map(update = false, strip = true, map, s3)
s4 = metadata.map(update = false, strip = true, map, s4)

s3 = prepend(s3, fun(_) -> blank(duration=4.))
#s3 = blank.strip(s3)
s1 = cue_cut(s1)
s1 = blank.eat(start_blank=true, threshold=-60., s1)
def post_meta()
    log("Send metadata to db")

    metadata = json()

    metadata.add('music', music_now_playing())
    metadata.add('field', field_recording_now_playing())
    metadata.add('vocal', vocal_now_playing())

    metadata.add('music_url', music_now_url())
    metadata.add('field_url', field_now_url())
    metadata.add('vocal_url', vocal_now_url())

    mt = json.stringify(metadata)

    ignore(http.post(headers=[("Content-Type","application/json")],data=mt,"https://millicent.org/api/backgroundplaylists"))
end

def track_metadata(m) =
  def api_post() =
    def remove_dotted_values(l) =
      def filter(m)
        let (key, value) = m
        not string.contains(substring=".", key) or string.contains(substring=".", value)
      end
      try
        list.filter(filter, l)
      catch err do
        log("Error in list filter - probably no data - #{error.kind(err)} - #{error.message(err)}")
        l
      end
    end
    
    m = remove_dotted_values(m)

    j = json()

    title = string.replace(pattern="'", fun(_) -> "", m['title'])
    #title = r/"'"/g.replace("", m['title'])

    timestamp = int_of_float(float_of_string(m['on_air_timestamp']) * 1000.)

    j.add('title',  title)
    j.add('artist', m['artist'])
    j.add('file_id', m['file_id'])
    j.add('on_air', m['on_air'])
    j.add('on_air_timestamp', timestamp)
    j.add('loudness', m['loudness'])
    j.add('type', m['type'])
    j.add('source_url', m['source_url'])

    dt = json.stringify(j)
    #dt = json.stringify(m)
    log("#{dt}")

8    res = http.post(headers=[("Content-Type","application/json")],data=dt,"https://millicent.org/api/vocalplaylists")

    #log("json #{dt}")

    #log("Post to databse of #{m['title']} returned - #{res.status_code}")
    if res.status_code > 400 then
      log("...... Database metadata post FAILED !!! #{res.status_code}")
    else
      log("...... Post to db")
    end
####
    if m['type'] == "music" then thread.run(fast=true,{music_playlog.add(m)})
    elsif m['type'] == "noise" then thread.run(fast=true,{field_playlog.add(m)})
    elsif m['type'] == "vocal" then thread.run(fast=true,{vocal_playlog.add(m)})
    else thread.run(fast=true,{v_fx_playlog.add(m)}) end

    log("s1 : #{s1type()}, words: #{words()}, s2 : #{s2type()}, s3 : #{s3type()}, s4 : #{s4type()}")
    thread.run(fast=true, post_meta)
  end

  if (m == []) then
    log("Null metadata passing through")
  #  meta(m)
  else
    thread.run(fast=false, delay=0.5, api_post)
  end

end

fx_timing_spb = ref(2.5)

## get beat from s1
def s1_bpm_call(ct) =
    s1bpm := ct
end

s1 = bpm(s1)
s2 = bpm(s2)
#p = bpm(poems)

##if s1 bpm is close to s2 bpm match it
def s2_bpm_set(ct) =
  f = 1. / (s1.bpm() / ct)

  if 0.2 <= f and f < 0.3 then
    s2bpm := f * 4.
  elsif 0.275 <= f and f < 0.4 then
    s2bpm := f * 3.
  elsif 0.40 < f and f < 0.60 then
    s2bpm := f  * 2.
  elsif 0.8 <= f and f <= 1.2 then
    s2bpm := f
  elsif 1.8 <= f and f <= 2.2 then
    s2bpm := f / 2.
  else
    s2bpm := 1.0
  end
# move s2bpm setter here
end

def gpt_talk(m)
    poems.add(nexttxt())
    poem_play := false
end

#thread.run(every=2.0,{s1_bpm_call(((s1.bpm())))})
thread.run(every=1.5,{if s1.bpm() != 0. then fx_timing_spb := (60. / s1.bpm() * 24.) end})
thread.run(fast=true, every=2.0,{s2_bpm_set(((s2.bpm())))})
# changed s2 to s3 above and s3 bpm above that
def event_trigger(t,m)
  log("Event triggered")
#  log("vox play - #{vox_play()}, v_fx_play - #{v_fx_play()}, recs playing - #{s2type()} - vox play - #{s3type()}, words - #{words()}")
  s2_q.add(nextb())
  def all_off()
    field_play := false
#    vox_play := false
    v_fx_play := false
  end

  def v_fx_push()
#    def log_track(stream)
#        log("track added to #{stream} queue")
#    end
#    log_track("field recording")
#    s2_q.set_queue(nextb())
### remove gpt poetry for now
#    if poem_play() then
      #elsif poem_play() then
      #gpt_talk(m)
#      poems.set_queue(nexttxt())
#      poem_play := false
#      field_play := false
      #end
    if vox_play() then
      s3_q.add(nextv())
      notPlayAlone := true
      ambient_only := true
      loud := "3"
      vox_play := false
    elsif v_fx_play() then
      s4_q.add(nextvfx())
      v_fx_play := false
    else
      all_off()
    end
  end
  if not words() and not s3type() then
      thread.run(v_fx_push)
  else
      thread.run(fast=true, delay=2., all_off)
  end
end


#s1 = cue_cut(s1)
#s1.on_metadata(gpt_talk)

s1 = on_offset(event_trigger, override="liq_on_offset_2", s1)

def set_meta_mus(m)
    id = m["id"]
    time = time.string("%Y-%m-%dT%H:%M:%S.000Z")
    log(time)
    data = json()
    data.add("lastPlayed", time)

    log("Setting music playing to #{m['title']}")
    music_now_playing := "#{m['title']} by #{m['artist']}"
    dt = json.stringify(data)
    res = http.put(headers=[("Authorization", "Basic " ^ auth() ),("Content-Type","application/json")],data=dt,"http://localhost:8080/content/#{id}")
    if res.status_code < 400 then
        log("music play logged")
    else
        log(res)
        log("FAILED TO LOG PLAYED FILE ON DB!")
    end
end

s1 = source.on_track(s1, set_meta_mus)

def default_dom_freq(default_val, val) =
  if val == "" then default_val else val end
end
######
# Smart transition for crossfade
# @category Source / Fade
# @param ~log Default logger
# @param ~fade_in  Fade-in duration, if any.
# @param ~fade_out Fade-out duration, if any.
# @param ~high     Value, in dB, for loud sound level.
# @param ~medium   Value, in dB, for medium sound level.
# @param ~margin   Margin to detect sources that have too different sound level for crossing.
# @param ~default Smart crossfade: transition used when no rule applies (default: sequence).
# @param a Ending track
# @param b Starting track
def cross.smart(~log=fun(x)->log(label="cross.smart",x),
                ~fade_in=3.,~fade_out=3.,
                ~default=(fun (a,b) -> (sequence([a, b]):source)),
                ~high=-15., ~medium=-32., ~margin=4.,
                a, b)
  def fade.out(s) = fade.out(type="sin",duration=fade_out,s) end
  def fade.in(s)  = fade.in(type="sin",duration=fade_in,s) end
  add = fun (a,b) -> add(normalize=false,[b, a])

  # This is for the type system..
  ignore(a.metadata["foo"])
  ignore(b.metadata["foo"])

  if
    # If A and B are not too loud and close, fully cross-fade them.
    a.db_level <= medium and b.db_level <= medium and abs(a.db_level - b.db_level) <= margin
    then
      log("Old <= medium, new <= medium and |old-new| <= margin.")
      log("Old and new source are not too loud and close.")
      log("Transition: crossed, fade-in, fade-out.")
      add(fade.out(a.source),fade.in(b.source))

  elsif
    # If B is significantly louder than A, only fade-out A.
    # We don't want to fade almost silent things, ask for >medium.
    b.db_level >= a.db_level + margin and a.db_level >= medium and b.db_level <= high
  then
    log("new >= old + margin, old >= medium and new <= high.")
    log("New source is significantly louder than old one.")
    log("Transition: crossed, fade-out.")
    add(fade.out(a.source),b.source)

  elsif
    # Opposite as the previous one.
    a.db_level >= b.db_level + margin and b.db_level >= medium and a.db_level <= high
    then
    log("old >= new + margin, new >= medium and old <= high")
    log("Old source is significantly louder than new one.")
    log("Transition: crossed, fade-in.")
    add(a.source,fade.in(b.source))

  elsif
    # Do not fade if it's already very low.
    b.db_level >= a.db_level + margin and a.db_level <= medium and b.db_level <= high
  then
    log("new >= old + margin, old <= medium and new <= high.")
    log("Do not fade if it's already very low.")
    log("Transition: crossed, no fade.")
    add(a.source,b.source)

  # What to do with a loud end and a quiet beginning ?
  # A good idea is to use a jingle to separate the two tracks,
  # but that's another story.

  else
    # Otherwise, A and B are just too loud to overlap nicely, or the
    # difference between them is too large and overlapping would completely
    # mask one of them.
    log("No transition: using default.")
    default(a.source, b.source)
  end
end

def transition_music(a,b)
  eight_bars = fx_timing_spb() / 2.0
  if eight_bars > 15.0 then eight_bars = fx_timing_spb() / 4.0 end
#  sixteen_bars = eight_bars * 2.
#  two_bars = eight_bars / 2.
#  spacer = 1.
  #t2 = two_bars / 2.
#  space = two_bars
  log("fxtiming is #{fx_timing_spb()}, 8 Bars is #{eight_bars} Seconds! a replay gain is #{a.metadata['liq_amplify']}, b replay gain is #{b.metadata['liq_amplify']}")
#  if (b.metadata['mix_type'] == "Play alone") and (a.metadata['mix_type'] == "Play alone" or a.metadata['genre'] == "Ethnic" or a.metadata['genre'] == "Electro" or b.metadata['genre'] == "Electro" or b.metadata['genre'] == "Ethnic") then
#    log("Play alone going in and out - no cross")
#    sequence([a.source, b.source])
    #sequence([fade.out(type="lin", duration = eight_bars, a.source), blank(duration = space),fade.in(duration = two_bars, type="lin", b.source)])
#  elsif a.metadata['mix_type'] == 'Play alone' and b.metadata['genre'] == "Ambient" then
#    log("Long trnasition from play alone to ambient")
#    add(normalize=false, [fade.out(duration = eight_bars, a.source), fade.in(duration= eight_bars, b.source)])
#  elsif (b.metadata['genre'] == "Ambient") and (a.metadata['genre'] == "Ambient") then
#    log("x fade 8 beats")
#    ##add(normalize=false,[fade.out(type="lin", duration = eight_bars, a.source), sequence([blank(duration = spacer), fade.in(duration = eight_bars, type="lin", b.source)])])
#    add(normalize=false,[fade.out(type="lin", duration = eight_bars, a.source), fade.in(duration = eight_bars, type="lin", b.source)])
#  if (a.metadata['source'] == "music_q") then
#    log("transition to field on music")
#    add(normalize=false, [fade.out(duration = 10., a.source), fade.in(duration= 10., b.source)])
#  elsif (a.metadata['outro'] == "vocal") or (b.metadata['intro'] == "vocal") then
#    log("no fade ")
#    sequence([a.source, b.source])
#  if not s3type() then
  cross.smart(fade_in=eight_bars, fade_out=eight_bars, a, b)
#  else
#    log("8 beat fade out gap fade in ")
#    sequence([fade.out(type="lin", duration = eight_bars, a.source), blank(duration = spacer),fade.in(duration = two_bars, type="lin", b.source)])
    #add([fade.out(duration=two_bars, a.source), fade.in(duration=two_bars, b.source)])
#  end
end
#
##music
def dur()
  if music_push_q_length() > 0 then
    fx_timing_spb() / 1.5
  else
   fx_timing_spb() / 1.5
  end
end

s1 = cross(duration=dur, transition_music, s1)

##field recordings

#s2.on_track(track_metadata)
s2_notch_eq_freq = interactive.float("field_notch_eq(hz)", description="Field notch freq", min=0., max=1000.0, 261.626)
s2_notch_eq_q = interactive.float("field_notch_eq_q", description="Field notch freq q", min=0.1, max=3.0, 2.)
s2_notch_eq_gain = interactive.float("field_notch_eq_gain", description="Field notch freq gain", min=-12., max=12.0, -6.)
8
s2 = filter.iir.eq.peak(frequency=float_of_string(music_centroid()), q=s2_notch_eq_q, gain=s2_notch_eq_gain, s2)

s2_lim_thresh = interactive.float("field limiter threshold (dB)", description="Field limiter threshold", min=-60., max=0.0, 0.0)
s2_lim_pre_gain = interactive.float("field limiter pre-gain (dB)", description="Field limiter pre-gain", min=-30., max=10., 0.0)
#s2 = amplify(s2_lim_pre_gain, s2)
#s2 = limit(s2_lim_thresh, s2)
s2 = ladspa.fastlookaheadlimiter(limit=s2_lim_thresh, input_gain= s2_lim_pre_gain, release_time=.12, s2)
#s2 = filter.iir.eq.highshelf(frequency=7000.0, slope=-3.0, s2)
s2 = lv2.bauer_stereophonic_to_binaural(s2)
#s2 = ladspa.lsp_limiter_stereo(threshold=s2_lim_thresh, input_gain=s2_lim_pre_gain, oversampling=10, s2)
##############s2 lufs
#s2 = lv2.zamheadx2(elev=-45., az=0., s2)
#s2 = lufs(s2)
#thread.run(every=1., fun() -> log("#{s2.lufs()}"))
#s2 = soundtouch(tempo = {s2bpm()}, s2)

def map_blank(m)
    #field_recording_now_playing := "silence by silence"
    [
    ("title", "silence"),
    ("artist", "silence")
    ]
end
silence_f = blank() 
silence_f = metadata.map(update = false, strip = true, map_blank, silence_f)
def skip_s2()
  if silence_f.is_ready() then
    if words() then
      log("Skiping field recording")
      s2_q.set_queue([])
      s2_q.skip()
    end
  end
end

def s2_leave(_,_)
  log("Field recording ending.")
#  field_recording_now_playing := "silence by silence"
#  thread.run(fast=false, post_meta)
  s2type := false
end

#s2 = source.on_end(s2, s2_leave)

s2 = switch(track_sensitive=false, transition_length=10.,[({words()},silence_f),({true}, s2)])

def set_meta_s2(m)
    log("Setting field playing to #{m['title']}")
    field_recording_now_playing := "#{m['title']} by #{m['artist']}"
end

s2 = source.on_metadata(s2, set_meta_s2)

def s2_transition(a,b)
  if b.metadata['title'] == "silence" then 
    field_recording_now_playing := "silence by silence"
    s2type := false
  end
  add(normalize=false, [fade.out(duration = 10., a.source), fade.in(duration= 10., b.source)])
end
s2 = cross(duration=10., s2_transition, s2)

#s2 = soundtouch(tempo = {s2bpm()}, s2)

#s2_switch = mksafe(s2_switch)

def vocal_end(_,_)
  notPlayAlone := true
  s3type := false
  ambient_only := false
#  vocal_now_playing := "silence by silence"
#  log("Vocal ending - s3type is #{s3type()}")
#  thread.run(fast=false, post_meta)
  #meta([])
end
##vocal
def set_meta_vox(m)
    log("Setting vocal playing to #{m['title']}")
    vocal_now_playing := "#{m['title']} by #{m['artist']}"
end

#s3 = source.on_end(s3, vocal_end)
s3 = source.on_metadata(s3, set_meta_vox)
#s3 = soundtouch(tempo = {s2bpm()}, s3)
s3 = ladspa.sc4(attack_time=24., release_time=450., knee_radius=1.0, ratio=4., makeup_gain=1.0, threshold_level=-6., s3)
#s3.on_track(track_metadata)
#s3 = blank.strip(max_blank=2.0, s3)
#added below line
#s3 = ladspa.sc4(attack_time=15., release_time=32., ratio=4.0, makeup_gain=0.0, threshold_level=-14.5, s3)
#s3 = ladspa.valve(distortion_character = 0.1, distortion_level = 0.1, s3)
#s3 = ladspa.tap_deesser(frequency={8373.1},threshold_level={-23.}, s3)
##s3 = ladspa.matrixspatialiser(width={4}, s3)
#s3 = ladspa.lsp_limiter_stereo(operating_mode=4, dithering=8, oversampling=10, gain_boost=false, s3)
#s3 = source.on_end(s3, vocal_end)
#def s3_transition(a,b)
#  #field_recording_now_playing := "silence by silence"
#  if b.metadata["source"] != "vocal" then
#      s3type := false
#      log("Setting s3type to #{s3type()}")
#  end
#  sequence([a.source, b.source])
#end

#s3= cross(s3_transition, s3)

def vocal_fx_end(t, m)
  s4type := false
  log("Vocal fx ending - s4type is #{s4type()}")
end

##vocal fx
s4 = append(s4,fun(_)->blank(duration=3.0))
s4 = filter.iir.eq.low_high(low={263.74}, high={8373.1}, s4)
s4 = ladspa.tapedelay(dry_level = {-18.0},tap_1_distance = {fx_timing_spb()},tap_1_level = {-12.0} , tap_2_distance = {fx_timing_spb() * 2.}, tap_2_level = {-19.0}, tap_3_distance = {fx_timing_spb() * 4.}, tap_3_level = {-16.0}, tap_4_distance = {fx_timing_spb() * 6.}, tap_4_level = {-4.0}, tape_speed={1.0}, s4)
#s4 = ladspa.matrixspatialiser(width={96}, s4)
##s4 = ladspa.zita_reverb(id="verb4", delay = {0.069}, output_mix=0.6, f1_freq = {2530.}, f1_gain = {2.474}, f2_freq = {7530.}, f2_gain = {-2.68},rt_low = {!fx_timing_spb * 0.75}, rt_mid = {!fx_timing_spb}, s4)
#s4 = ladspa.tap_reverb(decay=!fx_timing_spb, s4)

s4 = source.on_end(s4, vocal_fx_end)
def poems_bpm()
    fx_timing_spb() * 200.
end
#poems = soundtouch(tempo= {s2bpm()}, poems)
#poems = ladspa.zamdelay(dry_wet=0.2, feedback=1., poems)
#poems = filter.iir.eq.lowshelf(frequency=1000.0, slope=-3.0, poems)
#poems = lv2.bauer_stereophonic_to_binaural(poems)
#poems = ladspa.tap_reverb(decay=!fx_timing_spb * 10., poems)
#s1 = buffer(s1)

##### Add stream metadata for hls stream.
s1 = insert_metadata(s1)

thread.run(fast=true, every=1., {
    s1.insert_metadata([("title","*#{music_now_playing()}*#{vocal_now_playing()}*#{field_recording_now_playing()}")])
    })


music_mix_ratio = interactive.float("Music mix", description="Music mix", min=0.0, max=2.0, 1.0)
field_mix_ratio = interactive.float("Field mix", description="Field mix", min=0.0, max=2.0, 1.0)
vocal_mix_ratio = interactive.float("Voacl mix", description="Vocal mix", min=0.0, max=2.0, 1.0)
vocal_fx_mix_ratio = interactive.float("Vocal Fx mix", description="Vocal Fx mix", min=0.0, max=2.0, 1.0)

#s1 = source.run(s1, every=10., {print("#{profiler.stats.string()}")})

#-------------
# Mixes two streams, with faded transitions between the state when only the
# normal stream is available and when the special stream gets added on top of
# it.
# @category Source / Fade
# @flag extra
# @param ~duration Duration of the fade in seconds.
# @param ~p       Portion of amplitude of the normal source in the mix.
# @param ~normal  The normal source, which could be called the carrier too.
# @param ~special The special source.
def smooth_add(~duration=1., ~p=getter(0.2), ~normal, ~special)
  #duration = fx_timing_spb()
  p = getter.function(p)
  last_p = ref(p())

  def c(fn,s) =
    def v() =
      fn = fn()
      fn()
    end
    fade.scale(v,s)
  end

  special_volume = ref(fun () -> 0.)
  special = c(special_volume,special)

  normal_volume = ref(fun () -> 1.)
  normal = c(normal_volume,normal)

  def to_special(_,special) =
    s3type := true
    last_p := p()
    q = 1. - last_p()
#    ambient_only := true
    normal_volume := mkfade(start=1.,stop=last_p(),duration=duration,normal)
    special_volume := mkfade(stop=q,duration=duration,special)
    special
  end

  def to_blank(special,b)
    s3type := false
    notPlayAlone := false
    ambient_only := false
    vocal_now_playing := "silence by silence"
    log("s3type set to #{s3type()} - vox end")
    normal_volume := mkfade(start=last_p(),duration=duration,normal)
    special_volume := mkfade(start=1.-last_p(),duration=duration,special)
    sequence([special,b])
  end

  special = fallback(track_sensitive=false,
               transitions=[to_special,to_blank],
               [special,blank()])

  add(normalize=false,[normal,special])
end
#-------------------------
#s1 = smooth_add(duration=3.0, p=0.75, normal=s1, special=s3)
submix = add(normalize=false, weights=[music_mix_ratio, field_mix_ratio, vocal_fx_mix_ratio, vocal_mix_ratio], [s1, s2, s4, poems])
submix = smooth_add(duration=3.0, p=0.30, normal=submix, special=s3)
#submix = smooth_add(duration=1.0, p=0.85, normal=submix, special=poems)

#def b() =
#  log("BLANK")
#  #if (not !s1type) and (not !s2type) then
#  #  #print("Field Recordings skipped due to blank detection")
#  #  s2_q.skip()
#  #end
#end

#mix = blank.detect(id="blank", track_sensitive=false, max_blank=5.0, threshold=-45., b, mix)

thr = interactive.float("threshold (dB)", description="Main compressor threshold", min=-20., max=0., -14.)
gain = interactive.float("gain (dB)", description="Main compressor gain", min=-10., max=15., 0.)
rat = interactive.float("ratio (x:1)", description="Main compressor ratio", min=-0., max=6., 1.8)
att = interactive.float("attack (ms)", description="Main compressor attack", min=-0., max=100., 20.)
rel = interactive.float("release (ms)", description="Main compressor release", min=-0., max=1500., 20.)
knee = interactive.float("knee", description="Main compressor knee radius", min=-0., max=10., 4.25)

mb_pre = ref(1.0)
multiband_pre_gain = interactive.float("Multiband pre gain (dB)", description="Multiband pre gain", min=-10., max=20., 1.)
thread.run(every=10.,{mb_pre := multiband_pre_gain()})
submix = amplify({mb_pre()}, override="no", submix)

#mb_thres = interactive.float("mb_thres", description="mb_comp",min=-20.,max=0., -6.)
#mb_att = interactive.float("mb_att", description="mb_comp_att",min=0.,max=100., 20.0)
#mb_rel = interactive.float("mb_rel", description="mb_comp_rel",min=0.,max=2000., 100.0)
#
#submix =  ladspa.lsp_mb_compressor_stereo(compression_mode_0=1,
#                                       compression_mode_1=1,
#                                       compression_mode_2=1,
#                                       compression_mode_3=1,
#                                       compression_mode_4=1,
#                                       compression_mode_5=1,
#                                       compression_mode_6=1,
#                                       compression_mode_7=1,
#                                       attack_threshold_0 = mb_thres,
#                                       attack_threshold_1 = mb_thres,
#                                       attack_threshold_2 = mb_thres,
#                                       attack_threshold_3 = mb_thres,
#                                       attack_threshold_4 = mb_thres,
#                                       attack_threshold_5 = mb_thres,
#                                       attack_threshold_6 = mb_thres,
#                                       attack_threshold_7 = mb_thres,
#                                       attack_time_0 = mb_att,
#                                       attack_time_1 = mb_att,
#                                       attack_time_2 = mb_att,
#                                       attack_time_3 = mb_att,
#                                       attack_time_4 = mb_att,
#                                       attack_time_5 = mb_att,
#                                       attack_time_6 = mb_att,
#                                       attack_time_7 = mb_att,
#                                       release_time_0 = mb_rel,
#                                       release_time_1 = mb_rel,
#                                       release_time_2 = mb_rel,
#                                       release_time_3 = mb_rel,
#                                       release_time_4 = mb_rel,
#                                       release_time_5 = mb_rel,
#                                       release_time_6 = mb_rel,
#                                       release_time_7 = mb_rel,
#                                       ratio_0 = 1.8,
#                                       ratio_1 = 1.2,
#                                       ratio_2 = 1.3,
#                                       ratio_3 = 1.5,
#                                       ratio_4 = 1.7,
#                                       ratio_5 = 1.6,
#                                       ratio_6 = 1.5,
#                                       ratio_7 = 1.6,
#                                       submix)
#
submix = ladspa.sc4(attack_time=att, release_time=rel, knee_radius=knee, ratio=rat, makeup_gain=gain, threshold_level=thr, submix)
#submix = lv2.lsp_phase_detector(sel=50., submix)
submix = ladspa.lsp_limiter_stereo(operating_mode=2, oversampling=10, gain_boost=true, submix)

#def skip_field()
#  if !words and !s2type then
#    s2_q.set_queue([])
#    s2_q.skip()
#  end
#end

#thread.run(every=2.,skip_field)
#submix = insert_metadata(id="main_metadata", submix)
#
#def update_metadata(m)
#  submix.insert_metadata([("title", "#{!music_now_playing}*#{!vocal_now_playing}*#{!field_recording_now_playing}")])
#end
##mix.on_track(update_metadata)
#submix = source.on_metadata(submix, update_metadata)
#mix.insert_metadata([("title", "millicent - Sound Mirror - #{!music_now_playing}*#{!vocal_now_playing}*#{!field_recording_now_playing}*full ¬ silence by silence")])
#def print_m(m)
#    print(m)
#end
#submix = source.on_metadata(submix, print_m)
#mix = buffer(mksafe(mix))
#mix = buffer(max=30., mksafe(mix))
###### log output volume
#submix = lufs(submix)
#thread.run(every=2.0, fun() -> print(submix.lufs()))
#clock.assign_new(id="icecast", [submix])

####hls ouput
#aac_lofi = %ffmpeg(format="mpegts",
#                   %audio(
#                    codec="aac",
#                    channels=2,
#                    ar=48000
#                   )).{id3.version=3}
#
#aac_midfi = %ffmpeg(format="mpegts",
#                    %audio(
#                      codec="aac",
#                      channels=2,
#                      ar=48000,
#                      b="96k"
#                    )).{id3.version=3}
#
#aac_hifi = %ffmpeg(format="mpegts",
#                   %audio(
#                     codec="aac",
#                     channels=2,
#                     ar=48000,
#                     b="256k"
#                   )).{id3.version=3}
#
#streams = [("aac_lofi",aac_lofi), 
#           ("aac_midfi", aac_midfi), 
#           ("aac_hifi", aac_hifi)]
#                
def segment_name(~position,~extname,stream_name) =
  timestamp = int_of_float(time())
  duration = 6
  "#{stream_name}_#{duration}_#{timestamp}_#{position}.#{extname}"
end

output.file.hls(playlist="live.m3u8",
                fallible=true,
                segment_duration=6.0,
                segments=5,
                segments_overhead=5,
                segment_name=segment_name,
                persist_at="/tmp/hls/state.config",
                "/tmp/hls",
                [
                  ("aac_lofi",
                      %ffmpeg(format="mpegts", %audio(codec="aac", channels=2, ar=48000)).{
                        id3_version=3
                        }),
                  ("aac_midfi",
                      %ffmpeg(format="mpegts", %audio(codec="aac", channels=2, ar=48000, b="96k")).{
                        id3_version=3
                        }),
                  ("aac_hifi",
                      %ffmpeg(format="mpegts", %audio(codec="aac", channels=2, ar=48000, b="256k")).{
                        id3_version=3
                        }),
                ],
                submix)

#output.icecast(fallible=true, description="Sound Mirror", format="audio/aac", %fdkaac, host="ec2-63-34-87-186.eu-west-1.compute.amazonaws.com", port=8000, password="R@d1o5",mount="test",submix)
#output.icecast(fallible=true, description="Sound Mirror", %mp3, host="ec2-63-34-87-186.eu-west-1.compute.amazonaws.com", port=8000, password="R@d1o5",mount="test",mix)

interactive.persistent("./comp.params")

#### setup push requests for music and field recordings streams at :8000/mpush and :8000/fpush where {"file":"location of file" (e.g. s3://<bucket_name>/<fname>)}
#
#def mpush(~protocol,~data,~headers,uri)
#
#  let metadata.json.parse ( d : [(string * string)] as json.object) = data
#
#  s1_q.push.uri(d['file'])
#  
#  res = json()
#  code = ref(0)
#
#  if s1_q.length() > !music_push_q_length then
#    print("file added to main q")
#    music_push_q_length := s1_q.length()
#    res.add("msg", "file added to music queue!")
#    code := 200
#  else
#    res.add("msg", "FAILED to add file to music queue!")
#    code := 404
#  end
#
#  let json.stringify res = res
#
#  # Return response
#  http.response(
#   protocol=protocol,
#   code=!code,
#   headers=[("Content-Type","application/json; charset=utf-8"), ("Access-Control-Headers-Allow","*")],
#   data=res
#  )
#end
#
#def fpush(~protocol,~data,~headers,uri)
#
#  let json.parse ( d : [(string * string)] as json.object) = data
#
#  s2_q_push.push.uri(d['file'])
#  
#  res = json()
#  code = ref(0)
#
#  if s2_q_push.length() > !field_push_q_length then
#    print("file added to field recordings q")
#    field_push_q_length := s2_q_push.length()
#    res.add("msg", "file added to field recordings queue!")
#    code := 200
#  else
#    res.add("msg", "FAILED to add file to field recordings queue!")
#    code := 404
#  end
#
#  let json.stringify res = res
#
#  # Return response
#  http.response(
#   protocol=protocol,
#   code=!code,
#   headers=[("Content-Type","application/json; charset=utf-8"), ("Access-Control-Headers-Allow","*")],
#   data=res
#  )
#end
#
#harbor.http.register(port=7000,method="POST","/mpush", mpush)
#harbor.http.register(port=7000,method="POST","/fpush", fpush)
# Return the json content of meta


#def get_meta(request, response) =
#  s3_q.set_queue(nexttxt())

  #s2_q_push.push.uri(d['text'])
#  response.html("<html><body><b>done</b></body></html>")
#end

# Register get_meta at port 700
#harbor.http.register(port=7000,method="GET","/getmeta",get_meta)


