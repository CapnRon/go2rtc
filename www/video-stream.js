import {VideoRTC} from './video-rtc.js';

/**
 * This is example, how you can extend VideoRTC player for your app.
 * Also you can check this example: https://github.com/AlexxIT/WebRTC
 */
class VideoStream extends VideoRTC {
    set divMode(value) {
        this.querySelector('.mode').innerText = value;
        this.querySelector('.status').innerText = '';
    }

    set divError(value) {
        const state = this.querySelector('.mode').innerText;
        if (state !== 'loading') return;
        this.querySelector('.mode').innerText = 'error';
        this.querySelector('.status').innerText = value;
    }

    /**
     * Custom GUI
     */
    oninit() {
        console.debug('stream.oninit');
        super.oninit();

        this.innerHTML = `
        <style>
        video-stream {
            position: relative;
        }
        .info {
            position: absolute;
            top: 0;
            left: 0;
            right: 0;
            padding: 12px;
            color: white;
            display: flex;
            justify-content: space-between;
            pointer-events: none;
        }
        .sei {
            position: absolute;
            inset: 0;
            pointer-events: none;
        }
        .sei-el {
            position: absolute;
            color: white;
            font: 16px/1.3 sans-serif;
            text-shadow: 0 0 2px #000, 0 0 2px #000, 0 0 2px #000, 0 0 2px #000;
            white-space: pre;
        }
        </style>
        <div class="info">
            <div class="status"></div>
            <div class="mode"></div>
        </div>
        <div class="sei"></div>
        `;

        const info = this.querySelector('.info');
        this.insertBefore(this.video, info);

        this.seiContainer = this.querySelector('.sei');
        this.seiRAF = 0;
    }

    onconnect() {
        console.debug('stream.onconnect');
        const result = super.onconnect();
        if (result) this.divMode = 'loading';
        return result;
    }

    ondisconnect() {
        console.debug('stream.ondisconnect');
        super.ondisconnect();

        if (this.seiRAF) {
            cancelAnimationFrame(this.seiRAF);
            this.seiRAF = 0;
        }
        this.seiValue = null;
        this.seiAnchors = null;
        if (this.seiContainer) this.seiContainer.innerHTML = '';
    }

    onopen() {
        console.debug('stream.onopen');
        const result = super.onopen();

        this.onmessage['stream'] = msg => {
            console.debug('stream.onmessge', msg);
            switch (msg.type) {
                case 'error':
                    this.divError = msg.value;
                    break;
                case 'mse':
                case 'hls':
                case 'mp4':
                case 'mjpeg':
                    this.divMode = msg.type.toUpperCase();
                    break;
            }
        };

        return result;
    }

    onclose() {
        console.debug('stream.onclose');
        return super.onclose();
    }

    onpcvideo(ev) {
        console.debug('stream.onpcvideo');
        super.onpcvideo(ev);

        if (this.pcState !== WebSocket.CLOSED) {
            this.divMode = 'RTC';
        }
    }

    /**
     * Thingino SEI OSD overlay. Note: this does NOT rotate the video itself
     * to match `value.rotation` - it only positions elements within the
     * video's current (already displayed) box.
     * @param {Object} value - `{rotation, elements: [{t, text, x, y}, ...]}`
     */
    onsei(value) {
        this.seiValue = value;

        // anchor a "videoTime -> base Date" pair per timestamp element index,
        // so we can interpolate a smoothly ticking clock between updates
        this.seiAnchors = (value.elements || []).map(el => {
            if (el.t !== 'timestamp') return null;
            const base = new Date(el.text.replace(' ', 'T'));
            if (isNaN(base.getTime())) return null;
            return {base, videoTime: this.video.currentTime};
        });

        if (!this.seiRAF) this.seiRAF = requestAnimationFrame(() => this.renderSEI());
    }

    renderSEI() {
        this.seiRAF = 0;

        const value = this.seiValue;
        const container = this.seiContainer;
        if (!container) return;

        if (!value || !value.elements || !value.elements.length) {
            container.innerHTML = '';
            return;
        }

        const vw = this.video.videoWidth, vh = this.video.videoHeight;
        if (!vw || !vh) {
            // metadata not loaded yet, retry next frame
            this.seiRAF = requestAnimationFrame(() => this.renderSEI());
            return;
        }

        while (container.children.length > value.elements.length) {
            container.removeChild(container.lastChild);
        }
        while (container.children.length < value.elements.length) {
            container.appendChild(document.createElement('div'));
        }

        value.elements.forEach((el, i) => {
            const div = container.children[i];
            div.className = 'sei-el';

            let text = el.text || '';
            const anchor = this.seiAnchors && this.seiAnchors[i];
            if (anchor) {
                const offsetMs = (this.video.currentTime - anchor.videoTime) * 1000;
                text = formatTimestamp(new Date(anchor.base.getTime() + offsetMs));
            }
            div.textContent = text;

            const x = el.x || 0, y = el.y || 0;
            div.style.top = div.style.bottom = div.style.left = div.style.right = '';
            div.style.transform = '';

            if (y > 0) {
                div.style.top = (y / vh * 100) + '%';
            } else if (y < 0) {
                div.style.bottom = (-y / vh * 100) + '%';
            } else {
                div.style.top = '50%';
                div.style.transform = 'translateY(-50%)';
            }

            if (x > 0) {
                div.style.left = (x / vw * 100) + '%';
            } else if (x < 0) {
                div.style.right = (-x / vw * 100) + '%';
            } else {
                div.style.left = '50%';
                div.style.transform += ' translateX(-50%)';
            }
        });

        this.seiRAF = requestAnimationFrame(() => this.renderSEI());
    }
}

function formatTimestamp(date) {
    const pad = n => String(n).padStart(2, '0');
    return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} `
        + `${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`;
}

customElements.define('video-stream', VideoStream);
