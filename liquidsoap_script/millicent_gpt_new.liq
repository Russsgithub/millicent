
#settings.decoder.file_extensions.taglib.set([""])

## Tweak polly protocol to include -key (musical key of voice 0 = cmaj 12 = cmin amaj =4 24 = bmin) and other var (r = rate) or parsody

settings.init.allow_root.set(true)
settings.frame.audio.samplerate := 48000
settings.scheduler.fast_queues := 6
settings.scheduler.generic_queues := 6
settings.scheduler.non_blocking_queues := 6

#settings.scheduler.queues := [("generic", 6), ("non_blocking", 6), ("autocue", 6)]

#settings.autocue.internal.queues_count := 6

settings.log.level := 3

## for use with liq 2.2.5 +
#settings.autocue.amplify_behavior := "keep"
settings.autocue.internal.max_overlap := 6.
settings.autocue.internal.sustained_endings_enabled := true
settings.autocue.internal.cue_out_threshold := -48.0
settings.autocue.internal.cue_in_threshold := -40.0

# Autocue causing issues with processing lock up , try s3 on switch 
# prefetch 1 and or use protocol in api response which only cals autocue on music stream.
enable_autocue_metadata()
#enable_replaygain_metadata()

interactive.harbor()

s1bpm = ref(1.0)
s2bpm = ref(1.0)

s1type = ref(false)
s2type = ref(false)
s3type = ref(false)
s4type = ref(false)

s2_verb = ref(false)

words = ref(false)

loud = ref("4")
notPlayAlone = ref(true)
spec_band_norm = ref("6")
ambient_only = ref(false)

music_centroid = ref("800.0")

now_playing = json()

def init_metadata_json ()
    now_playing.add("music", {title = "silence", artist = "silence", source_url = "silence", image = null()})
    now_playing.add("field", {title = "silence", artist = "silence", source_url = "silence"})
    now_playing.add("vocal", {title = "silence", artist = "silence", source_url = "silence"})
end

init_metadata_json()

music_now_url = ref("")
field_now_url = ref("")
vocal_now_url = ref("")

music_push_q_length = ref(0)
field_push_q_length = ref(0)

music_offset_trigger = ref(0.0)

music_playlog = playlog(duration=3600., persistency="./music_playlog", hash=fun (m) -> file.digest(metadata.filename(m)))
field_playlog = playlog(duration=3600., persistency="./field_playlog", hash=fun (m) -> file.digest(metadata.filename(m)))
vocal_playlog = playlog(duration=3600., persistency="./vocal_playlog", hash=fun (m) -> file.digest(metadata.filename(m)))
v_fx_playlog = playlog(duration=3600., persistency="./vfx_playlog", hash=fun (m) -> file.digest(metadata.filename(m)))

next_track_decision_key = interactive.string(description="Metadata key to calculate next track from. (norm_centroid or norm_energy)", "next track decision key", "norm_centroid")

last_music_mix_type = ref("")

last_radio_metadata = ref([])

play_vox_now = ref(false)

#### vocal trigger flags
v_fx_play = ref(false)
def play_v_fx()
  log("allowing effected vocal to play")
  v_fx_play := true
end
thread.when({25m or 55m},play_v_fx)

vox_play = ref(false)
def play_vox()
  log("allowing vocal to play")
  vox_play := true
#  ambient_only := true
end
thread.when({15m},play_vox)
#thread.when({55m or 05m or 15m or 25m or 35m or 45m},play_vox)

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

def is_dusk_dawn_calc()
    if time.local().hour == 20 or time.local().hour == 7 then
        dusk_dawn := true
    else
        dusk_dawn := false
    end
end

thread.when({0m}, is_dusk_dawn_calc)

auth = ref(string.base64.encode("control:m1ll1c3nt"))

attempt_count_music = ref(0)

play_alone_consecutive_count = ref(0)

a_o = interactive.bool(description="ao", "Ambient only", false)

#thread.run(every=5.0, fun() -> print("#{a_o()}"))

def next_track(stream)
  base_url = "https://millicent.org/next"
  next_track_url = ref("#{base_url}?stream=#{stream}")

  print("#{a_o()}")
  

  if stream == "music" then
    nPA = if notPlayAlone() or play_alone_consecutive_count() > 1 then "Mix" else "" end
    ao = if ambient_only() or dusk_dawn() or a_o() then "Ambient" else "" end
    next_track_url := "#{next_track_url()}&#{next_track_decision_key()}=#{loud()}&mix_type=#{nPA}&genre=#{ao}"
  end

  result = http.get(headers=[("Authorization", "Basic " ^ auth() )], next_track_url())

  if string.contains(substring="zero Choices with Weight", result) and attempt_count_music() == 0 then
    attempt_count_music := attempt_count_music() + 1
    ambient_only := false
    dusk_dawn := false
  end

#  result = if stream != "music" then "annotate:liq_disable_autocue=true:"^result else result end

  request.create("annotate:liq_blankskip=0.00:"^result)
end


s1_q = request.dynamic(id="music_alone", prefetch=3, {next_track("music")})
#s1_q = request.dynamic(id="music_q", prefetch=0, nextm_b)

s2_q = request.dynamic(id="field", prefetch=0, retry_delay=10., {next_track("noise")})

s3_q = request.dynamic(id="vocal", prefetch=1, retry_delay=10., {next_track("vocal")})

s4_q = request.dynamic(id="vocal_fx", prefetch=0, retry_delay=10.0, {next_track("vocal_fx")})

#s1 = fallback([s1_q, s1])
s2_fallback = s2_q

s1 = amplify(id="alone_amp", override="liq_amplify", 1.0, s1_q)
s2 = amplify(id="field_amp", override="liq_amplify", 0.22, s2_fallback)
s3 = amplify(id="vox_amp", override="liq_amplify", 1.2, s3_q)
s4 = amplify(id="vox_fx_amp", override="liq_amplify", 0.4, s4_q)

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
  print(m)
  type = m['stream']
  title = m['title']
  artist = m['artist']
  source_url = m['source_url']
  space = if artist != "" then " " else "" end
  source_url_fallback = "https://duckduckgo.com/?q="^artist^space^title

  ### next track decision key variable, set above map function.
  loudness = default("5", m[next_track_decision_key()])
  # spec_band_norm_val = default("5", m['spec_bandwidth_norm'])
  log("playing - #{title} on #{type}")

  def increase_music_loud_val()
    loud := string(int_of_string(loud()) + 1)
    log("loud equals #{loud()}")
  end

  if type == "music" then
    attempt_count_music := 0
    if music_playlog.last(m) < 3600. then
      if s3type() then
          log("vocal")
      else
          log("repeat")
      end
      log("skipping music - either vocal is playing and music choice is not Amb or a repeat has been stopped")
#      thread.run(increase_music_loud_val)
      s1_q.skip()
      s1type := false
      words := false
      []
    elsif m['mix_type'] == "Play alone" then
      if s3type() then
        log("skipping music - vocals are playing and play alone has been selected")
        s1_q.skip()
        s1type := false
        words := false
        []
      else
        log("Words set to true")
        offset = m['liq_on_offset']
        music_offset_trigger := if offset != "" then
            float_of_string(offset)
        else
            0.0
        end
        log("offset: #{music_offset_trigger()}")
        url = if source_url == "" then source_url_fallback else source_url end
        music_now_url := "#{url}"
        words := true
        s1type := true
        loud := loudness
        play_alone_consecutive_count := play_alone_consecutive_count() + 1
        music_playlog.add(m)
#        spec_band_norm := spec_band_norm_val
#        notPlayAlone := true
#        music_centroid := default("800.0", m["dominant_frequency"])
        #music_now_playing := "#{title} by #{artist}"
#        last_music_key := m['key']
        m
      end
    else
      log("Words set to false")
      offset = m['liq_on_offset']
      music_offset_trigger := if offset != "" then
          float_of_string(offset)
      else
          0.0
      end
      log("offset: #{music_offset_trigger()}")
      url = if source_url == "" then source_url_fallback else source_url end
      music_now_url := "#{url}"
      words := false
      s1type := true
      notPlayAlone := false
      loud := loudness
      play_alone_consecutive_count := 0
      music_playlog.add(m)
#      spec_band_norm := spec_band_norm_val
#      music_centroid := default("800.0", m["dominant_frequency"])
      #music_now_playing := "#{title} by #{artist}"
#      last_music_key := m['key']
      m
    end
  elsif type == "noise" then
    print("Preparing field recording")
    if int_of_string(loudness) > 5 then
      s2_verb := true
    else
      s2_verb := false
    end

    if words() or (field_playlog.last(m) < 3600.) then
      log("skipping field rec - play alone playing or repeat")
      s2type := false
      []
    else
      url = if source_url == "" then source_url_fallback else source_url end
      field_now_url := "#{url}"
      #field_recording_now_playing := "#{title} by #{artist}"
      s2type := true
      m
    end
  elsif type == "vocal" then
    if words() then
      log("skipping vocal - words on music")
#      s3_q.set_queue([])
      play_vox_now := false
      s3_q.skip()
      s3type := false
      ambient_only := false
      notPlayAlone := false
      []
    else
      url = if source_url == "" then source_url_fallback else source_url end
      vocal_now_url := "#{url}"
      #vocal_now_playing := "#{title} by #{artist}"
      #loud := "-1"
#      s3type := true
      ambient_only := true
      play_vox_now := false
      m
    end
  elsif type == "vocal_fx" then
    if words() or s3type() then
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

s1 = metadata.map(update = false, strip = true, map, s1)
s2 = metadata.map(update = false, strip = true, map, s2)
s3 = metadata.map(update = false, strip = true, map, s3)
s4 = metadata.map(update = false, strip = true, map, s4)

s3 = prepend(s3, fun(_) -> blank(duration=4.))

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

thread.run(every=1.0,{if s1.bpm() != 0. then fx_timing_spb := (60. / s1.bpm() * 24.) end})
#thread.run(fast=true, every=2.0,{s2_bpm_set(((s2.bpm())))})
# changed s2 to s3 above and s3 bpm above that

vocal_queued = ref(false)

def event_trigger(t,m)
  log("Event triggered")
#  log("vox play - #{vox_play()}, v_fx_play - #{v_fx_play()}, recs playing - #{s2type()} - vox play - #{s3type()}, words - #{words()}")
  if list.is_empty(s2_q.queue()) then
    log("Adding field recoding to queue")
    def p_s2()
      s2_q.add(next_track("noise"))
      ()
    end
    thread.run(delay=fx_timing_spb(), p_s2)
    true
  else
    log("NOT adding to filed queue")
    false
  end

  def all_off()
    log("ALL off!")
    vox_play := false
    v_fx_play := false
    vocal_queued := false
  end

  def v_fx_push()
    stream = ref("vocal_fx")

    def track_push()
      log("Queuing #{stream()}.")
      all_off()
      if stream() == "vocal" then
        s3_q.add(next_track("vocal"))
      else
        s4_q.add(next_track("vocal_fx"))
      end
      ()
    end

    if vox_play() then
      print("Can play vocals, attempting")
      vocal_queued := true
      notPlayAlone := true
      #ambient_only := true
      loud := ""
      vox_play := false
      stream := "vocal"

      delay = source.remaining(s1) + fx_timing_spb()
      log("Pushing vocal in #{delay} seconds")      
      thread.run(delay=delay, fun() -> play_vox_now := true)
      # Remove last item in music queue
      #music_queue = s1_q.queue()
      #extra_music_request = list.last(default=null(), music_queue)
      #request.destroy(extra_music_request)
      ##
      #request.destroy(list.last(s1_q.queue()))
      #s1_q.set_queue([])
      #delay = source.remaining(s1)
    elsif not words() and v_fx_play() then
      v_fx_play := false
      stream := "vocal_fx"
      delay = fx_timing_spb() * 2.
      log("Pushing vocal_fx in #{delay} seconds")
      thread.run(delay=delay, track_push)
    else
      notPlayAlone := false
      ambient_only := false
      all_off()
    end
  end

  if not words() and not s3type() then
      log("Getting ready to run cmd push.")
      thread.run(v_fx_push)
  else
      log("play alone music playing, skipping vocals this time round.")
      thread.run(all_off)
  end
end

s1 = on_offset(offset = music_offset_trigger(), event_trigger, s1)

def save_cover(m)
  print("Trying to Extract Cover")
  cvr = metadata.cover(m)
  t = m['id']
  fn = "/home/ubuntu/millicent/golang_api/static/covers/#{t}.jpeg"

  cover_url = if cvr != "" and (m['coverart'] != "" or m['pic'] != "" or m['apic'] != "") then  
    if not file.exists(fn) then
      log("Writting file #{fn}")
      ignore (file.write(data=cvr, fn))
    end
    "https://millicent.org/static/covers/#{t}.jpeg"
  else
    log("No cover art found in metadata")
    log("Using default image for cover art")
    "https://millicent.org/static/covers/default_cover.png"
  end

  cover_url
end

def update_db(m)
    id = m["id"]
    playcount = if m['play_count'] != "" then string(int_of_string(m['play_count']) + 1) else "1" end
    data = json()
    data.add("last_played", time.string("%Y-%m-%dT%H:%M:%S.000Z"))
    data.add("play_count", playcount)

    dt = json.stringify(data)
    if id != "" then
      ignore(http.put(headers=[("Authorization", "Basic " ^ auth() ),("Content-Type","application/json")],data=dt,"https://millicent.org/content/#{id}"))
    else
      print("No id present in metadata for: #{m['title']}")
      print(m)
    end
end

def set_meta_mus(m)
    if m['stream'] == "music" then 
      cover_url = save_cover(m)
      
      now_playing.add("music", {title = "#{m['title']}", artist = "#{m['artist']}", source_url = "#{music_now_url()}", image = cover_url })
    end
    ## add track to json metadata
    if m['mix_type'] == "Play alone" then 
        log("Music tagged as play alone, skipping s2 track playing")        
        s2_q.set_queue([])
        s2_q.skip()
    end
end

s1 = source.on_metadata(s1, set_meta_mus)
s1 = source.on_track(s1, update_db)

def default_dom_freq(default_val, val) =
  if val == "" then default_val else val end
end

### Crossfade for autocue ### Use one of the other not both.

s1 = crossfade(s1)

### Or custom crossfade

######
### Smart transition for crossfade
### @category Source / Fade
### @param ~log Default logger
### @param ~fade_in  Fade-in duration, if any.
### @param ~fade_out Fade-out duration, if any.
### @param ~high     Value, in dB, for loud sound level.
### @param ~medium   Value, in dB, for medium sound level.
### @param ~margin   Margin to detect sources that have too different sound level for crossing.
### @param ~default Smart crossfade: transition used when no rule applies (default: sequence).
### @param a Ending track
### @param b Starting track
##def cross.smart(~log=fun(x)->log(label="cross.smart",x),
##  ~fade_in=3.,~fade_out=3.,
##  ~default=(fun (a,b) -> (sequence([a, b]):source)),
##  ~high=-15., ~medium=-32., ~margin=4.,
##  a, b)
##def fade.out(s) = fade.out(type="sin",duration=fade_out,s) end
##def fade.in(s)  = fade.in(type="sin",duration=fade_in,s) end
##add = fun (a,b) -> add(normalize=false,[b, a])
##
### This is for the type system..
##ignore(a.metadata["foo"])
##ignore(b.metadata["foo"])
##
##if
### If A and B are not too loud and close, fully cross-fade them.
##a.db_level <= medium and b.db_level <= medium and abs(a.db_level - b.db_level) <= margin
##then
##log("Old <= medium, new <= medium and |old-new| <= margin.")
##log("Old and new source are not too loud and close.")
##log("Transition: crossed, fade-in, fade-out.")
##add(fade.out(a.source),fade.in(b.source))
##
##elsif
### If B is significantly louder than A, only fade-out A.
### We don't want to fade almost silent things, ask for >medium.
##b.db_level >= a.db_level + margin and a.db_level >= medium and b.db_level <= high
##then
##log("new >= old + margin, old >= medium and new <= high.")
##log("New source is significantly louder than old one.")
##log("Transition: crossed, fade-out.")
##add(fade.out(a.source),b.source)
##
##elsif
### Opposite as the previous one.
##a.db_level >= b.db_level + margin and b.db_level >= medium and a.db_level <= high
##then
##log("old >= new + margin, new >= medium and old <= high")
##log("Old source is significantly louder than new one.")
##log("Transition: crossed, fade-in.")
##add(a.source,fade.in(b.source))
##
##elsif
### Do not fade if it's already very low.
##b.db_level >= a.db_level + margin and a.db_level <= medium and b.db_level <= high
##then
##log("new >= old + margin, old <= medium and new <= high.")
##log("Do not fade if it's already very low.")
##log("Transition: crossed, no fade.")
##add(a.source,b.source)
##
### What to do with a loud end and a quiet beginning ?
### A good idea is to use a jingle to separate the two tracks,
### but that's another story.
##
##else
### Otherwise, A and B are just too loud to overlap nicely, or the
### difference between them is too large and overlapping would completely
### mask one of them.
##log("No transition: using default.")
##default(a.source, b.source)
##end
##end


s1 = source.drop.metadata(s1)

#def transition_music(a,b)
#eight_bars = fx_timing_spb() / 2.0
#if eight_bars > 15.0 then eight_bars = fx_timing_spb() / 4.0 end
##log("fxtiming is #{fx_timing_spb()}, 8 Bars is #{eight_bars} Seconds! a replay gain is #{a.metadata['liq_amplify']}, b replay gain is #{b.metadata['liq_amplify']}")
#cross.smart(fade_in=eight_bars, fade_out=eight_bars, a, b)
#end
##
###music
#def dur()
#if music_push_q_length() > 0 then
#fx_timing_spb() / 1.5
#else
#fx_timing_spb() / 1.5
#end
#end
#
#s1 = cross(duration=dur, transition_music, s1)

### End of custom crossfade

##field recordings

s2_notch_eq_freq = interactive.float("field_notch_eq(hz)", description="Field notch freq", min=0., max=5000.0, 261.626)
s2_notch_eq_q = interactive.float("field_notch_eq_q", description="Field notch freq q", min=0.1, max=3.0, 2.)
s2_notch_eq_gain = interactive.float("field_notch_eq_gain", description="Field notch freq gain", min=-12., max=12.0, -6.)

s2_lim_pre_gain = interactive.float("field limiter pre-gain (dB)", description="Field limiter pre-gain", min=-0., max=2., 0.25)

s2 = amplify(s2_lim_pre_gain, override="none", s2)

s2 = filter.iir.eq.peak(frequency=s2_notch_eq_freq, q=s2_notch_eq_q, gain=s2_notch_eq_gain, s2)

s2 = ladspa.sc4(attack_time=1.0, release_time=1000.0, knee_radius=1.0, ratio=2.0, makeup_gain=1.0, threshold_level=-6., s2)

s2 = ladspa.bs2b(s2)

def update_field_metadata(m)
  now_playing.add("field", {title = "#{m['title']}", artist = "#{m['artist']}", source_url = "#{field_now_url()}"})
end

s2 = source.on_track(s2, update_field_metadata)
# beat sync field recs to music
#s2 = soundtouch(tempo = {s2bpm()}, s2)

s2 = source.on_track(s2, update_db)

def map_blank(m)
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

s2 = switch(track_sensitive=false, transition_length=15., [({words()},silence_f),({true}, s2)])

def s2_transition(a,b)
  dur =  fx_timing_spb()
  dur = if dur < 5.0 then
    dur * 4.
  elsif dur < 10. then
    dur * 2.
  elsif dur > 20. then
    dur / 2.
  else
    dur
  end

  if b.metadata['title'] == "silence" or list.is_empty(b.metadata) then 
    now_playing.add("field", {title = "silence", artist = "silence", source_url = ""})
    s2type := false
  else
    now_playing.add("field", {title = "#{b.metadata['title']}", artist = "#{b.metadata['artist']}", source_url = "#{field_now_url()}"})
  end
  add(normalize=false, [fade.out(duration = dur, a.source), fade.in(duration= dur, b.source)])
end

s2 = cross(duration=12., s2_transition, s2)

##vocal
s3 = switch(track_sensitive=true, [({play_vox_now()}, s3)])

s3 = ladspa.sc4(attack_time=24., release_time=450., knee_radius=1.0, ratio=4., makeup_gain=1.0, threshold_level=-6., s3)

s3 = source.on_metadata(s3, fun(m) -> now_playing.add("vocal", {title = "#{m['title']}", artist = "#{m['artist']}", source_url = "#{vocal_now_url()}"}))

s3 = source.on_track(s3, update_db)


##vocal fx
def vocal_fx_end(t, m)
  s4type := false
  log("Vocal fx ending - s4type is #{s4type()}")
end

# editted duration from 4. to beat calc
s4 = append(s4,merge=true, fun(_)->blank(duration=fx_timing_spb() * 10))
s4 = filter.iir.eq.low_high(low={263.74}, high={8373.1}, s4)
s4 = ladspa.tapedelay(dry_level = {-18.0},tap_1_distance = {fx_timing_spb()},tap_1_level = {-12.0} , tap_2_distance = {fx_timing_spb() * 2.}, tap_2_level = {-19.0}, tap_3_distance = {fx_timing_spb() * 4.}, tap_3_level = {-16.0}, tap_4_distance = {fx_timing_spb() * 8.}, tap_4_level = {-6.0}, tape_speed={1.0}, s4)

s4 = source.on_end(s4, vocal_fx_end)

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
    log("s3type set to #{s3type()} - vox start")
    last_p := p()
    q = 1. - last_p()
    normal_volume := mkfade(start=1.,stop=last_p(),duration=duration,normal)
    special_volume := mkfade(stop=q,duration=duration,special)
    special
  end

  def to_blank(special,b)
    s3type := false
    notPlayAlone := false
    ambient_only := false
    now_playing.add("vocal", {title = "silence", artist = "silence", source_url = ""})
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

music_mix_ratio = interactive.float("Music mix", description="Music mix", min=0.0, max=2.0, 1.0)
field_mix_ratio = interactive.float("Field mix", description="Field mix", min=0.0, max=2.0, 1.0)
vocal_mix_ratio = interactive.float("Voacl mix", description="Vocal mix", min=0.0, max=2.0, 1.0)
vocal_fx_mix_ratio = interactive.float("Vocal Fx mix", description="Vocal Fx mix", min=0.0, max=2.0, 1.0)

main_mix = add(normalize=true, weights=[music_mix_ratio, field_mix_ratio, vocal_fx_mix_ratio], [s1, s2, s4])

submix = smooth_add(duration=5.0, p=0.4, normal=main_mix, special=s3)
#submix = main_mix
##### Add stream metadata for hls stream.

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

submix = ladspa.sc4(attack_time=att, release_time=rel, knee_radius=knee, ratio=rat, makeup_gain=gain, threshold_level=thr, submix)
#submix = ladspa.lsp_limiter_stereo(operating_mode=2, oversampling=10, gain_boost=false, submix)

# submix = compress.exponential(mu=1.0, submix)
submix = normalize(target=-14.0, gain_min=-6.0, gain_max=6.0, lufs=true, lookahead=0.1,  submix)
submix = limit(threshold=-3.0, ratio=15.0, submix)

seg_dur = ref(10)  
 
def segment_name(metadata) =
  timestamp = int_of_float(time())
  let {stream_name, duration, position, extname} = metadata
  "#{stream_name}_#{duration}_#{timestamp}_#{position}.#{extname}"
end

def push_to_s3()
  process.run("aws s3 sync /tmp/hls/ s3://millicent-audio-mirror/hls --delete")
end

def on_file_change(~state, fname)
    if state == "updated" and file.extension(fname) == ".m3u8" then
        thread.run(push_to_s3)
    end
end


submix = insert_metadata(submix)

thread.run(every=0.5, {submix.insert_metadata([("title", json.stringify(now_playing))])})
#on_file_change=on_file_change,
#clock(submix)

#submix = lufs(submix)
#thread.run(every=1.0, fun() -> print("lufs: #{submix.lufs()}"))


output.file.hls(playlist="live.m3u8",
                segment_duration=float_of_int(seg_dur()),
                segments=7,
                segments_overhead=1,
                segment_name=segment_name,
                persist_at="/tmp/hls/state.config",
                perm=0o755,
                on_file_change=on_file_change,
                fallible=true,
                "/tmp/hls",
                [
                  #("aac_lofi",
                  #    %ffmpeg(format="mpegts", %audio(codec="aac", channels=2, ar=48000)).{
                  #      id3_version=4
                  #      }),
                  ("aac_hifi",
                      %ffmpeg(format="mpegts", %audio(codec="aac", channels=2, ar=48000, b="320k")).{
                        id3_version=4
                        })
                ],
                submix)

interactive.persistent("./comp.params")

def metadata(_)
http.response(data=json.stringify(now_playing))
end
harbor.http.register.simple(port=8000, "/metadata", metadata)

