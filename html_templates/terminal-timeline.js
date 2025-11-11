// Terminal Timeline Report JavaScript
let currentFrame = 0;
let isPlaying = false;
let playInterval;
let totalFrames = 0;
let testDurationMs = 0;
let scaledFrameTime = 0;
// frameData is provided by the HTML template

function initializeApp(framesCount, durationMs, frames) {
    console.log('🚀 initializeApp called with:', framesCount, 'frames, duration:', durationMs);
    console.log('🎬 frames parameter type:', typeof frames, 'length:', frames ? frames.length : 'UNDEFINED');
    console.log('📊 frames[0]:', frames && frames[0] ? frames[0].label : 'NO FRAME 0');

    // Show debug info directly in the UI
    const mainDisplay = document.getElementById('mainDisplay');
    if (mainDisplay) {
        mainDisplay.innerHTML = `
            <div style="color: #00ff00; font-family: monospace; padding: 20px; background: #000;">
                <h2>🔧 DEBUG INFO</h2>
                <p>✅ JavaScript is running!</p>
                <p>📊 Total frames: ${framesCount}</p>
                <p>🎬 Frame data type: ${typeof frames}</p>
                <p>📁 Frame data length: ${frames ? frames.length : 'UNDEFINED'}</p>
                <p>🎯 Frame 0: ${frames && frames[0] ? frames[0].label : 'NO FRAME 0'}</p>
                <p>🎯 Frame 4: ${frames && frames[4] ? frames[4].label : 'NO FRAME 4'}</p>
            </div>
        `;
    }

    totalFrames = framesCount;
    testDurationMs = durationMs;
    scaledFrameTime = Math.max(300, Math.floor((testDurationMs * 10) / totalFrames));
    frameData = frames;

    console.log('🔧 After assignment - frameData:', typeof frameData, 'length:', frameData ? frameData.length : 'UNDEFINED');
    console.log('🔧 frameData[0]:', frameData && frameData[0] ? frameData[0].label : 'NO frameData[0]');

    // Start the timeline automatically after initialization
    if (totalFrames > 0) {
        console.log('🚀 FORCING FRAME 4 FOR TESTING');
        // Don't call selectFrame yet, just show the debug info first
        setTimeout(() => {
            selectFrame(4);  // Force frame 4 which should have rich ANSI content
        }, 3000);
    }
}

function selectFrame(index) {
    console.log('Selecting frame', index);
    currentFrame = index;

    // Update timeline frame highlighting
    document.querySelectorAll('.timeline-frame').forEach(frame => {
        frame.classList.remove('active');
    });
    document.querySelector(`[data-index="${index}"]`).classList.add('active');

    // Update main display
    const mainDisplay = document.getElementById('mainDisplay');
    const frameTitle = document.getElementById('currentFrameTitle');
    const timeCounter = document.getElementById('timeCounter');

    if (frameData[index]) {
        console.log('Frame data available for index', index, 'label:', frameData[index].label);
        frameTitle.textContent = `Frame ${index}: ${frameData[index].label}`;

        // Show raw content first for debugging
        console.log('Raw frame content:', frameData[index].content.substring(0, 300));
        mainDisplay.innerHTML = frameData[index].content;

        // Initialize terminal emulator for ANSI content
        if (frameData[index].isANSI) {
            console.log('Initializing terminal for frame', index, 'with content preview:', frameData[index].content.substring(0, 200));
            initializeTerminal(index);
        } else {
            console.log('Frame', index, 'is not marked as ANSI content');
        }

        // Update time counter
        timeCounter.textContent = `${frameData[index].timing}ms`;

        // Add fallback test - show something no matter what
        setTimeout(() => {
            if (mainDisplay.querySelector('.terminal-screen') && mainDisplay.querySelector('.terminal-screen').innerHTML === '') {
                console.log('Terminal screen is empty, adding fallback content');
                const fallbackHTML = '<div style="color: #ffd43b; padding: 20px;">🔧 FALLBACK: Terminal content should appear here</div>';
                mainDisplay.querySelector('.terminal-screen').innerHTML = fallbackHTML;
            }
        }, 500);
    } else {
        console.error('No frame data for index', index);
    }

    // Auto-scroll film strip to keep current frame centered (like a real projector)
    const timelineStrip = document.getElementById('timelineStrip');
    const currentFrameElement = document.querySelector(`[data-index="${index}"]`);
    if (currentFrameElement && timelineStrip) {
        const stripRect = timelineStrip.getBoundingClientRect();
        const frameRect = currentFrameElement.getBoundingClientRect();
        const stripCenter = stripRect.height / 2;
        const frameCenter = frameRect.top - stripRect.top + (frameRect.height / 2);
        const scrollOffset = frameCenter - stripCenter;

        timelineStrip.scrollTo({
            top: timelineStrip.scrollTop + scrollOffset,
            behavior: 'smooth'
        });
    }
}

function initializeTerminal(frameIndex) {
    // Find terminal containers in the main display area only
    const mainDisplay = document.getElementById('mainDisplay');
    if (!mainDisplay) {
        console.log('No mainDisplay found!');
        return;
    }
    const terminalContainers = mainDisplay.querySelectorAll('.terminal-container');
    console.log('Found', terminalContainers.length, 'terminal containers in mainDisplay');

    terminalContainers.forEach((container, containerIndex) => {
        const ansiContent = container.getAttribute('data-ansi-content');
        console.log('Container', containerIndex, 'has ANSI content:', ansiContent ? ansiContent.length + ' chars' : 'NONE');
        if (!ansiContent) return;

        const terminalScreen = container.querySelector('.terminal-screen');
        if (!terminalScreen) {
            console.log('No terminal-screen found in container', containerIndex);
            return;
        }

        // Clear any existing terminal
        terminalScreen.innerHTML = '';

        // Check if xterm.js is available
        if (typeof Terminal === 'undefined' || typeof FitAddon === 'undefined') {
            console.log('xterm.js not available, using fallback for ANSI content:', ansiContent.substring(0, 100));
            const terminalHTML = createSimpleANSIDisplay(ansiContent);
            terminalScreen.innerHTML = terminalHTML;
            console.log('Fallback ANSI display created, HTML length:', terminalHTML.length);
            return;
        }

        // Create new xterm.js terminal
        console.log('Creating xterm.js terminal for container', containerIndex);
        const terminal = new Terminal({
            cursorBlink: false,
            convertEol: true,
            fontFamily: 'Monaco, Menlo, "DejaVu Sans Mono", "Lucida Console", monospace',
            fontSize: 12,
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

        // Open terminal in the container
        terminal.open(terminalScreen);
        fitAddon.fit();

        // Convert escaped content back to actual ANSI sequences
        const unescapedContent = ansiContent
            // First handle Unicode escape sequences (like \u001b for ESC)
            .replace(/\\u([0-9a-fA-F]{4})/g, (match, hex) => {
                return String.fromCharCode(parseInt(hex, 16));
            })
            // Then handle standard HTML entities
            .replace(/&quot;/g, '"')
            .replace(/&#34;/g, '"')   // Additional quote encoding
            .replace(/&#39;/g, "'")
            .replace(/&lt;/g, '<')
            .replace(/&gt;/g, '>')
            .replace(/&amp;/g, '&')    // Keep this last to avoid double-unescaping
            .replace(/\\n/g, '\n')
            .replace(/\\r/g, '\r');

        // Write the ANSI content to the terminal
        console.log('Unescaped ANSI content sample:', unescapedContent.substring(0, 200));
        terminal.write(unescapedContent);

        // Disable input
        terminal.onData(() => {
            // Do nothing - this is read-only
        });
    });
}

// Simple ANSI-to-HTML converter (no external dependencies)
function createSimpleANSIDisplay(ansiContent) {
    console.log('Raw ANSI content received:', ansiContent.substring(0, 200));

    // Convert escaped content back to readable format (handle both standard and Unicode escaping)
    const unescapedContent = ansiContent
        // First handle Unicode escape sequences (like \u001b for ESC)
        .replace(/\\u([0-9a-fA-F]{4})/g, (match, hex) => {
            return String.fromCharCode(parseInt(hex, 16));
        })
        // Then handle standard HTML entities
        .replace(/&quot;/g, '"')
        .replace(/&#34;/g, '"')           // Additional quote encoding
        .replace(/&#39;/g, "'")
        .replace(/&lt;/g, '<')
        .replace(/&gt;/g, '>')
        .replace(/&amp;/g, '&')           // Keep this last to avoid double-unescaping
        .replace(/\\n/g, '\n')
        .replace(/\\r/g, '\r');

    console.log('After unescaping:', unescapedContent.substring(0, 200));

    // Basic ANSI escape sequence handling
    let displayContent = unescapedContent
        // Remove cursor control sequences
        .replace(/\x1b\[\d*[ABCD]/g, '') // cursor movement
        .replace(/\x1b\[\?\d*[hl]/g, '') // mode changes
        .replace(/\x1b\[\d*[JK]/g, '') // clear sequences
        .replace(/\x1b\[H/g, '') // home cursor
        .replace(/\x1b\[\?\d*h/g, '') // private mode
        .replace(/\x1b\[\?\d*l/g, '') // private mode
        .replace(/\x1b\[\d*;?\d*H/g, '') // cursor positioning
        // Simplify color codes (basic support)
        .replace(/\x1b\[([0-9;]+)m/g, (match, codes) => {
            // Basic color mapping for 256-color and standard codes
            const codeList = codes.split(';');
            let result = '';
            for (let i = 0; i < codeList.length; i++) {
                const code = parseInt(codeList[i]);
                if (code === 0) result += '</span></strong></em>'; // reset all
                else if (code === 1) result += '<strong>'; // bold
                else if (code === 3) result += '<em>'; // italic
                else if (code === 31) result += '<span style="color: #ff6b6b;">'; // red
                else if (code === 32) result += '<span style="color: #51cf66;">'; // green
                else if (code === 33) result += '<span style="color: #ffd43b;">'; // yellow
                else if (code === 34) result += '<span style="color: #74c0fc;">'; // blue
                else if (code === 35) result += '<span style="color: #f783ac;">'; // magenta
                else if (code === 36) result += '<span style="color: #3bc9db;">'; // cyan
                else if (code === 37) result += '<span style="color: #e9ecef;">'; // white
                else if (code === 39) result += '<span style="color: #e9ecef;">'; // default fg
                else if (code === 90) result += '<span style="color: #6c7086;">'; // bright black
                else if (code === 91) result += '<span style="color: #ff9580;">'; // bright red
                else if (code === 92) result += '<span style="color: #8cf59f;">'; // bright green
                else if (code === 93) result += '<span style="color: #ffe066;">'; // bright yellow
                else if (code === 94) result += '<span style="color: #a3c7ff;">'; // bright blue
                else if (code === 95) result += '<span style="color: #ff9fcc;">'; // bright magenta
                else if (code === 96) result += '<span style="color: #6cdaff;">'; // bright cyan
                else if (code === 97) result += '<span style="color: #ffffff;">'; // bright white
                else if (code === 38 && codeList[i+1] === '5') {
                    // 256-color support (basic mapping)
                    const colorCode = parseInt(codeList[i+2]);
                    i += 2; // skip next two codes
                    if (colorCode >= 16 && colorCode <= 231) {
                        // Calculate RGB for 256-color palette (simplified)
                        const r = Math.floor((colorCode - 16) / 36) * 51;
                        const g = Math.floor(((colorCode - 16) % 36) / 6) * 51;
                        const b = ((colorCode - 16) % 6) * 51;
                        result += `<span style="color: rgb(${r}, ${g}, ${b});">`;
                    } else if (colorCode >= 232 && colorCode <= 255) {
                        // Grayscale
                        const gray = (colorCode - 232) * 10 + 8;
                        result += `<span style="color: rgb(${gray}, ${gray}, ${gray});">`;
                    }
                }
            }
            return result;
        })
        // Convert newlines to HTML
        .replace(/\n/g, '<br>')
        .replace(/\r/g, '');

    console.log('Final processed content:', displayContent.substring(0, 300));

    const finalHTML = `
        <div style="
            background: #1a1a1a;
            color: #e9ecef;
            font-family: 'SF Mono', 'Monaco', 'Menlo', 'DejaVu Sans Mono', monospace;
            font-size: 12px;
            padding: 16px;
            border-radius: 4px;
            white-space: pre-wrap;
            overflow-x: auto;
            min-height: 200px;
            line-height: 1.4;
        ">
            <div style="margin-bottom: 8px; color: #ffd43b; font-size: 11px;">📺 Terminal Output</div>
            ${displayContent}
        </div>
    `;

    console.log('Generated HTML length:', finalHTML.length, 'chars');
    return finalHTML;
}

function nextFrame() {
    if (currentFrame < totalFrames - 1) {
        selectFrame(currentFrame + 1);
    }
}

function prevFrame() {
    if (currentFrame > 0) {
        selectFrame(currentFrame - 1);
    }
}

function toggleTimeline() {
    if (isPlaying) {
        clearInterval(playInterval);
        document.getElementById('playBtn').innerHTML = 'Play Film';
        document.getElementById('playBtn').classList.remove('playing');
        isPlaying = false;
    } else {
        playInterval = setInterval(() => {
            if (currentFrame < totalFrames - 1) {
                selectFrame(currentFrame + 1);
            } else {
                selectFrame(0); // Loop back to start
            }
        }, scaledFrameTime); // Based on actual test timing
        document.getElementById('playBtn').innerHTML = 'Stop Film';
        document.getElementById('playBtn').classList.add('playing');
        isPlaying = true;
    }
}

function resetTimeline() {
    if (isPlaying) {
        toggleTimeline();
    }
    selectFrame(0);
}

// Initialize on DOM content loaded
document.addEventListener('DOMContentLoaded', function() {
    // Auto-start is now handled in initializeApp() after data is loaded
});

// Keyboard navigation
document.addEventListener('keydown', function(e) {
    switch(e.key) {
        case 'ArrowRight':
            e.preventDefault();
            nextFrame();
            break;
        case 'ArrowLeft':
            e.preventDefault();
            prevFrame();
            break;
        case ' ':
            e.preventDefault();
            toggleTimeline();
            break;
        case 'Escape':
            if (isPlaying) {
                toggleTimeline();
            }
            break;
    }
});