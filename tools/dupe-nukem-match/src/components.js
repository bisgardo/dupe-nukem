customElements.define('custom-file', class extends HTMLElement {
    constructor() {
        super();
        this.attachShadow({mode: 'open'});
    }

    // noinspection JSUnusedGlobalSymbols
    connectedCallback() {
        const name = this.getAttribute('name')
        if (this.shadowRoot) this.shadowRoot.innerHTML = `
            <li part="name">${name}</li>
        `
    }
})

customElements.define('custom-dir', class extends HTMLElement {
    constructor() {
        super();
        this.attachShadow({mode: 'open'});
    }

    // noinspection JSUnusedGlobalSymbols
    connectedCallback() {
        const name = this.getAttribute('name')
        if (this.shadowRoot) this.shadowRoot.innerHTML = `
            <li>
                <span part="name">${name}</span>
                <ul part="children">
                    <slot></slot>
                </ul>
            </li>
        `
    }
})
