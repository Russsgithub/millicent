        // Function to initialize HLS.js and start the player
        function initializeAudioPlayer() {
            // Create an HLS.js instance
            var hls = new Hls();

            // Get a reference to the audio player element
            var audioPlayer = document.getElementById('audio-player');

            // Provide the URL of the HLS audio stream
            var streamUrl = 'https://dev.rapidcarrot.com/hls/live.m3u8';

            // Attach the HLS.js instance to the audio player
            hls.attachMedia(audioPlayer);

            // Load the HLS stream
            hls.loadSource(streamUrl);

            // Listen for events (optional)
            hls.on(Hls.Events.MANIFEST_PARSED, function () {
                // The stream is ready to play
                audioPlayer.play();
            });
        }

        // Call the initialization function when the page loads
        window.onload = function () {
            initializeAudioPlayer();
        };