/**
 * Components for managing the DOM representations of the domain types explicitly.
 * In the future, these classes could be folded back into the target classes to reduce the number of created objects.
 * DOM elements are connected using the methods `appendChild` and `appendTo` in a double-dispatch fashion:
 * The `appendChild` method is implemented by types that support having children, `appendTo` is supported by all nodes.
 * A node doesn't have any opinions on what kinds of elements can be children, only whether there can be any at all.
 * The idea is that a node always attaches itself to some child-supporting parent using `appendTo` by calling `appendChild` on the parent, passing its own root element.
 * As the method has the same signature as the one on `HTMLElement`, it doesn't matter if it's attaching to a plain DOM element or another node:
 * In the former case, it'll just be added directly. In the latter one, the called node will attach the caller to the appropriate container element.
 */

import {Target, Dir, File} from "./target"

/**
 * @type {WeakMap<HTMLElement, DirDom|FileDom>}
 */
const elements = new WeakMap()

export class DirDom {
    /**
     * @param {Dir} dir
     */
    constructor(dir) {
        this.dir = dir
        this.root = DirDom.#createRoot(dir.scanDir.name)
        this.container = this.root.appendChild(DirDom.#createContainer())
        elements.set(this.root, this)
    }

    /**
     * @param {string} name
     * @returns {HTMLLIElement}
     */
    static #createRoot(name) {
        const res = document.createElement('li')
        res.textContent = name
        return res
    }

    static #createContainer() {
        return document.createElement('ul')
    }

    /**
     * Append a child to the container.
     * @param {HTMLElement} child
     */
    appendChild(child) {
        this.container.appendChild(child)
    }

    /**
     * Append to the provided DOM element.
     * @param {TargetContainerDom|DirDom|HTMLElement} parent
     */
    appendTo(parent) {
        parent.appendChild(this.root)
    }

    /**
     * @param {boolean} v
     */
    setHighlighted(v) {
        if (v) {
            this.root.classList.add('highlighted')
        } else {
            this.root.classList.remove('highlighted')
        }
        return this
    }

    // TODO: Add method for expanding, collapsing etc.
}

export class FileDom {
    /**
     * @param {File} file
     */
    constructor(file) {
        this.file = file
        this.root = FileDom.#createRoot(file.scanFile.name)
        elements.set(this.root, this)
    }

    /**
     * @param {string} name
     * @returns {HTMLLIElement}
     */
    static #createRoot(name) {
        const res = document.createElement('li')
        res.textContent = name
        return res
    }

    /**
     * @param {boolean} v
     */
    setHighlighted(v) {
        if (v) {
            this.root.classList.add('highlighted')
        } else {
            this.root.classList.remove('highlighted')
        }
        return this
    }

    /**
     * Append to the provided DOM element.
     * @param {DirDom|HTMLElement} parent
     */
    appendTo(parent) {
        parent.appendChild(this.root)
    }
}

export class TargetContainerDom {
    /**
     * @param {Target} target
     */
    constructor(target) {
        this.target = target
        this.root = TargetContainerDom.#createRoot()
        this.root.addEventListener('mouseover', this.handleMouseOver)
        this.root.addEventListener('mouseout', this.handleMouseOut)
        this.container = this.root.appendChild(TargetContainerDom.#createContainer())

        /**
         * @type {DirDom|FileDom|null}
         */
        this.currentlyHighlighed = null
    }

    static #createRoot() {
        const root = document.createElement('div')
        root.className = 'target-container'
        return root
    }

    static #createContainer() {
        return document.createElement('ul')
    }

    /**
     * @param {DirDom|FileDom|null} dom
     */
    setHighlighed(dom) {
        if (this.currentlyHighlighed !== dom) {
            this.currentlyHighlighed?.setHighlighted(false)
        }
        if (dom) {
            dom.setHighlighted(true)
        }
        this.currentlyHighlighed = dom
    }

    // TODO: It feels like this logic should be moved to a controller of some sort.

    /**
     * @param {MouseEvent} e
     */
    handleMouseOver = (e) => {
        console.log('over', e.target)
        let nextHighlighted = null
        if (e.target instanceof HTMLElement) {
            const dom = elements.get(e.target);
            if (dom) nextHighlighted = dom;
        }
        this.setHighlighed(nextHighlighted)
    }

    /**
     * @param {MouseEvent} e
     */
    handleMouseOut = (e) => {
        console.log('out', e.target)
        this.setHighlighed(null)
    }

    /**
     * Append a child to the container.
     * @param {HTMLElement} child
     */
    appendChild(child) {
        this.container.appendChild(child)
    }

    /**
     * Append to the provided DOM element.
     * @param {HTMLElement} parent
     */
    appendTo(parent) {
        parent.appendChild(this.root)
    }
}
