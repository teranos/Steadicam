// Terminal State Player - Reconstructs actual terminal states from ANSI stream
let terminal = null;
let isPlaying = false;
let playbackPosition = 0;
let streamTimeout = null;
let playbackSpeed = 1.0;
let terminalStates = []; // Reconstructed terminal states

class TerminalState {
    constructor() {
        this.lines = [];
        this.cursorRow = 0;
        this.cursorCol = 0;
        this.width = 80;
        this.height = 24;

        // Initialize empty terminal
        for (let i = 0; i < this.height; i++) {
            this.lines.push(' '.repeat(this.width));
        }
    }

    // Write text at cursor position
    writeText(text) {
        if (this.cursorRow >= this.height) return;

        let line = this.lines[this.cursorRow];
        let newLine = line.substring(0, this.cursorCol) + text + line.substring(this.cursorCol + text.length);
        this.lines[this.cursorRow] = newLine.substring(0, this.width);
        this.cursorCol += text.length;
    }

    // Move cursor to position
    setCursor(row, col) {
        this.cursorRow = Math.max(0, Math.min(row, this.height - 1));
        this.cursorCol = Math.max(0, Math.min(col, this.width - 1));
    }

    // Clear screen
    clear() {
        for (let i = 0; i < this.height; i++) {
            this.lines[i] = ' '.repeat(this.width);
        }
        this.cursorRow = 0;
        this.cursorCol = 0;
    }

    // Get terminal content as text
    getContent() {
        return this.lines.join('\r\n');
    }

    // Create a copy of this state
    clone() {
        let newState = new TerminalState();
        newState.lines = [...this.lines];
        newState.cursorRow = this.cursorRow;
        newState.cursorCol = this.cursorCol;
        return newState;
    }
}

function initializeStatePlayer() {
    if (typeof streamEvents === 'undefined') {
        console.log('No stream events data found');
        return;
    }

    console.log('Reconstructing terminal states from stream...');
    reconstructTerminalStates();

    // Initialize xterm.js terminal
    terminal = new Terminal({
        cursorBlink: false,
        convertEol: true,
        fontFamily: 'Monaco, Menlo, "DejaVu Sans Mono", "Lucida Console", monospace',
        fontSize: 14,
        theme: {
            background: '#000000',
            foreground: '#ffffff',
            cursor: '#ffffff',
            selection: '#ffffff30'
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

    console.log(`State player initialized with ${terminalStates.length} terminal states`);
    updateControls();
}

function reconstructTerminalStates() {
    let currentState = new TerminalState();
    terminalStates = [];

    // Add initial empty state
    terminalStates.push({
        elapsed_ms: 0,
        description: 'Terminal initialized',
        content: currentState.getContent()
    });

    for (let i = 0; i < streamEvents.length; i++) {
        const event = streamEvents[i];

        if (event.event_type === 'output' && event.content) {
            // Parse and apply ANSI sequences to terminal state
            applyANSIToState(currentState, event.content);

            // Save this state
            terminalStates.push({
                elapsed_ms: event.elapsed_ms,
                description: event.description || `Terminal update ${i}`,
                content: currentState.getContent()
            });
        } else if (event.event_type === 'keypress') {
            // For keypress, we might want to show the character being typed
            // but usually this is handled by the output that follows
            console.log(`Keypress: ${event.description}`);
        } else if (event.event_type === 'marker') {
            if (event.description === 'REPL started') {
                currentState.clear();
            }
        }
    }

    console.log(`Reconstructed ${terminalStates.length} terminal states`);
}

function applyANSIToState(state, ansiContent) {
    // This is a simplified ANSI parser - in reality, you'd want a more complete one
    let content = ansiContent;

    // Remove problematic sequences
    content = content.replace(/\x1b\[\?1049[hl]/g, ''); // Alternate buffer
    content = content.replace(/\x1b\[\?25[lh]/g, ''); // Cursor visibility
    content = content.replace(/\x1b\[\?2004[hl]/g, ''); // Bracketed paste

    // Handle screen clear
    if (content.includes('\x1b[2J')) {
        state.clear();
        content = content.replace(/\x1b\[2J/g, '');
    }

    // Handle cursor home
    if (content.includes('\x1b[H')) {
        state.setCursor(0, 0);
        content = content.replace(/\x1b\[H/g, '');
    }

    // Handle cursor positioning (simplified)
    content = content.replace(/\x1b\[(\d+);(\d+)H/g, (match, row, col) => {
        state.setCursor(parseInt(row) - 1, parseInt(col) - 1);
        return '';
    });

    // Handle simple cursor movements
    content = content.replace(/\x1b\[(\d+)H/g, (match, row) => {
        state.setCursor(parseInt(row) - 1, 0);
        return '';
    });

    // Remove color codes but keep the text
    content = content.replace(/\x1b\[[0-9;]*m/g, '');

    // Handle carriage returns and newlines
    content = content.replace(/\r\n/g, '\n');
    content = content.replace(/\r/g, '\n');

    // Write remaining text
    if (content.trim()) {
        // Split by newlines and write each line
        const lines = content.split('\n');
        for (let i = 0; i < lines.length; i++) {
            if (lines[i]) {
                state.writeText(lines[i]);
            }
            if (i < lines.length - 1) {
                // Move to next line
                state.cursorRow++;
                state.cursorCol = 0;
            }
        }
    }
}

function playStateStream() {
    if (isPlaying) {
        stopStateStream();
        return;
    }

    isPlaying = true;
    updateControls();
    console.log(`Starting state playback from position ${playbackPosition}/${terminalStates.length}`);
    playNextState();
}

function playNextState() {
    if (!isPlaying || playbackPosition >= terminalStates.length) {
        stopStateStream();
        return;
    }

    const state = terminalStates[playbackPosition];
    const nextState = terminalStates[playbackPosition + 1];

    // Update progress
    updateProgress();

    // Clear and write the terminal state
    terminal.clear();
    terminal.write(state.content);

    console.log(`[${state.elapsed_ms}ms] ${state.description}`);

    playbackPosition++;

    // Calculate delay until next state
    if (nextState) {
        const delay = Math.max(50, (nextState.elapsed_ms - state.elapsed_ms) / playbackSpeed);
        streamTimeout = setTimeout(playNextState, delay);
    } else {
        // End of stream
        stopStateStream();
        console.log('State playback completed');
    }
}

function stopStateStream() {
    isPlaying = false;
    if (streamTimeout) {
        clearTimeout(streamTimeout);
        streamTimeout = null;
    }
    updateControls();
}

function resetStateStream() {
    stopStateStream();
    playbackPosition = 0;
    if (terminal && terminalStates.length > 0) {
        terminal.clear();
        terminal.write(terminalStates[0].content);
    }
    updateProgress();
    console.log('State stream reset to beginning');
}

function seekToStatePosition(position) {
    const wasPlaying = isPlaying;
    stopStateStream();

    playbackPosition = Math.max(0, Math.min(position, terminalStates.length - 1));

    if (terminal && terminalStates[playbackPosition]) {
        terminal.clear();
        terminal.write(terminalStates[playbackPosition].content);
    }

    updateProgress();

    if (wasPlaying && playbackPosition < terminalStates.length) {
        playStateStream();
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

    if (terminalStates.length === 0) return;

    const progress = (playbackPosition / terminalStates.length) * 100;
    const currentState = terminalStates[Math.min(playbackPosition, terminalStates.length - 1)];
    const totalTime = terminalStates[terminalStates.length - 1]?.elapsed_ms || 0;

    if (progressBar) {
        progressBar.style.width = progress + '%';
    }

    if (timeDisplay) {
        timeDisplay.textContent = `${currentState?.elapsed_ms || 0}ms / ${totalTime}ms`;
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
            playStateStream();
            break;
        case 'r':
            e.preventDefault();
            resetStateStream();
            break;
        case 'ArrowLeft':
            e.preventDefault();
            if (!isPlaying) {
                seekToStatePosition(playbackPosition - 5);
            }
            break;
        case 'ArrowRight':
            e.preventDefault();
            if (!isPlaying) {
                seekToStatePosition(playbackPosition + 5);
            }
            break;
    }
});

// Initialize when page loads
document.addEventListener('DOMContentLoaded', function() {
    console.log('Initializing terminal state player...');
    setTimeout(initializeStatePlayer, 100);
});