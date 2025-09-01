/**
 * Components for managing the DOM representations of the domain types explicitly.
 */

export class DirDom {
    /**
     * @param {string} name
     */
    constructor(name) {
        this.self = document.createElement('li')
        this.self.textContent = name
        this.container = this.self.appendChild(document.createElement('ul'))
    }

    /**
     * Append a child to the container.
     * @param {DirDom|FileDom} child
     */
    append(child) {
        this.container.appendChild(child.self)
    }

    // TODO: Add method for expanding, collapsing etc.
}

export class FileDom {
    /**
     * @param {string} name
     */
    constructor(name) {
        this.self = document.createElement('li')
        this.self.textContent = name
    }

    // TODO: Add methods for highlighting etc..
}