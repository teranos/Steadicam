// Stream Player for Continuous Terminal Output
let terminal = null;
let isPlaying = false;
let playbackPosition = 0;
let streamTimeout = null;
let playbackSpeed = 1.0;

function initializeStreamPlayer() {
    if (typeof streamEvents === 'undefined') {
        console.log('No stream events data found');
        return;
    }

    // Initialize xterm.js terminal
    terminal = new Terminal({
        cursorBlink: true,
        convertEol: true,
        fontFamily: 'Monaco, Menlo, "DejaVu Sans Mono", "Lucida Console", monospace',
        fontSize: 14,
        theme: {
            background: '#000000',
            foreground: '#ffffff',
            cursor: '#ffffff',
            selection: '#ffffff30',
            black: '#000000',
            red: '#ff6b6b',
            green: '#51cf66',
            yellow: '#ffd43b',
            blue: '#74c0fc',
            magenta: '#f783ac',
            cyan: '#3bc9db',
            white: '#ffffff'
        },
        cols: 80,
        rows: 24
    });

    // Create fit addon
    const fitAddon = new FitAddon.FitAddon();
    terminal.loadAddon(fitAddon);

    // Mount terminal to the screen
    const terminalScreen = document.getElementById('terminalScreen');
    terminal.open(terminalScreen);
    fitAddon.fit();

    // Disable input
    terminal.onData(() => {
        // Read-only terminal
    });

    console.log(`Stream player initialized with ${streamEvents.length} events`);
    updateControls();
}

function playStream() {
    if (isPlaying) {
        stopStream();
        return;
    }

    isPlaying = true;
    updateControls();

    console.log(`Starting playback from position ${playbackPosition}/${streamEvents.length}`);
    playNextEvent();
}

function playNextEvent() {
    if (!isPlaying || playbackPosition >= streamEvents.length) {
        stopStream();
        return;
    }

    const event = streamEvents[playbackPosition];
    const nextEvent = streamEvents[playbackPosition + 1];

    // Update progress
    updateProgress();

    // Handle the current event
    if (event.event_type === 'output' && event.content) {
        // Clean and process ANSI content before writing
        let cleanContent = cleanANSIForTerminal(event.content);
        if (cleanContent) {
            terminal.write(cleanContent);
            console.log(`[${event.elapsed_ms}ms] Terminal output: ${cleanContent.length} chars (cleaned from ${event.content.length})`);
        }
    } else if (event.event_type === 'keypress') {
        console.log(`[${event.elapsed_ms}ms] Keypress: ${event.description}`);
    } else if (event.event_type === 'marker') {
        console.log(`[${event.elapsed_ms}ms] Marker: ${event.description}`);

        // Clear terminal on REPL start to ensure clean state
        if (event.description === 'REPL started') {
            terminal.clear();
        }
    }

    playbackPosition++;

    // Calculate delay until next event
    if (nextEvent) {
        const delay = Math.max(10, (nextEvent.elapsed_ms - event.elapsed_ms) / playbackSpeed);
        streamTimeout = setTimeout(playNextEvent, delay);
    } else {
        // End of stream
        stopStream();
        console.log('Stream playback completed');
    }
}

function cleanANSIForTerminal(content) {
    if (!content) return '';

    // Remove problematic sequences that cause positioning issues
    let cleaned = content;

    // Remove alternate buffer commands (these mess up the display)
    cleaned = cleaned.replace(/\x1b\[\?1049[hl]/g, '');

    // Remove cursor visibility commands
    cleaned = cleaned.replace(/\x1b\[\?25[lh]/g, '');

    // Remove bracketed paste mode
    cleaned = cleaned.replace(/\x1b\[\?2004[hl]/g, '');

    // Handle screen clear more gracefully - convert to newlines instead
    cleaned = cleaned.replace(/\x1b\[2J/g, '\r\n\r\n');

    // Convert absolute cursor positioning to relative movements
    cleaned = cleaned.replace(/\x1b\[H/g, '\r\n');
    cleaned = cleaned.replace(/\x1b\[\d+;\d+H/g, '\r\n');

    // Clean up excessive carriage returns and newlines
    cleaned = cleaned.replace(/\r\n\r\n\r\n+/g, '\r\n\r\n');
    cleaned = cleaned.replace(/\r+/g, '\r');

    return cleaned;
}

function stopStream() {
    isPlaying = false;
    if (streamTimeout) {
        clearTimeout(streamTimeout);
        streamTimeout = null;
    }
    updateControls();
}

function resetStream() {
    stopStream();
    playbackPosition = 0;
    if (terminal) {
        terminal.clear();
    }
    updateProgress();
    console.log('Stream reset to beginning');
}

function seekToPosition(position) {
    const wasPlaying = isPlaying;
    stopStream();

    // Reset terminal and replay from beginning to target position
    if (terminal) {
        terminal.clear();
    }

    playbackPosition = 0;

    // Fast-forward through events up to target position
    while (playbackPosition < position && playbackPosition < streamEvents.length) {
        const event = streamEvents[playbackPosition];

        if (event.event_type === 'output' && event.content) {
            let cleanContent = cleanANSIForTerminal(event.content);
            if (cleanContent) {
                terminal.write(cleanContent);
            }
        }

        playbackPosition++;
    }

    updateProgress();

    // Resume playing if we were playing before
    if (wasPlaying && playbackPosition < streamEvents.length) {
        playStream();
    }
}

function updateControls() {
    const playBtn = document.getElementById('streamPlayBtn');
    const resetBtn = document.getElementById('streamResetBtn');
    const speedSelect = document.getElementById('speedSelect');

    if (playBtn) {
        playBtn.textContent = isPlaying ? 'Pause Stream' : 'Play Stream';
        playBtn.className = isPlaying ? 'control-btn playing' : 'control-btn';
    }

    if (resetBtn) {
        resetBtn.disabled = isPlaying;
    }

    if (speedSelect) {
        speedSelect.disabled = isPlaying;
    }
}

function updateProgress() {
    const progressBar = document.getElementById('streamProgress');
    const timeDisplay = document.getElementById('streamTime');

    if (typeof streamEvents === 'undefined' || streamEvents.length === 0) return;

    const progress = (playbackPosition / streamEvents.length) * 100;
    const currentEvent = streamEvents[Math.min(playbackPosition, streamEvents.length - 1)];
    const totalTime = streamEvents[streamEvents.length - 1]?.elapsed_ms || 0;

    if (progressBar) {
        progressBar.style.width = progress + '%';
    }

    if (timeDisplay) {
        timeDisplay.textContent = `${currentEvent?.elapsed_ms || 0}ms / ${totalTime}ms`;
    }
}

function changePlaybackSpeed(speed) {
    playbackSpeed = parseFloat(speed);
    console.log(`Playback speed changed to ${playbackSpeed}x`);
}

// Keyboard controls
document.addEventListener('keydown', function(e) {
    switch(e.key) {
        case ' ':
            e.preventDefault();
            playStream();
            break;
        case 'r':
            e.preventDefault();
            resetStream();
            break;
        case 'ArrowLeft':
            e.preventDefault();
            if (!isPlaying) {
                seekToPosition(Math.max(0, playbackPosition - 10));
            }
            break;
        case 'ArrowRight':
            e.preventDefault();
            if (!isPlaying) {
                seekToPosition(Math.min(streamEvents.length - 1, playbackPosition + 10));
            }
            break;
    }
});

// Initialize when page loads
document.addEventListener('DOMContentLoaded', function() {
    console.log('Initializing stream player...');
    setTimeout(initializeStreamPlayer, 100);
});