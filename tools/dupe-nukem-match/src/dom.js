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

// TODO:
//  - Both DirDom and FileDom currently have identical implementations of method 'mark'. If it stays that way, consider extracting a common base class.

import { Controller } from "./controller.js"
import {Dir, File, Target} from "./domain.js"

/** @type {WeakMap<HTMLElement, DirDom|FileDom>} */
export const domMap = new WeakMap()

export class DirDom {
    /**
     * @param {Dir} dir
     */
    constructor(dir) {
        this.dir = dir
        this.root = DirDom.#createRoot(dir.name)
        this.container = this.root.appendChild(DirDom.#createContainer())
        domMap.set(this.root, this)
    }

    /**
     * @param {string} name
     * @returns {HTMLElement}
     */
    static #createRoot(name) {
        const res = document.createElement('li')
        res.className = 'dir'
        const nameContainer = res.appendChild(document.createElement('div'))
        nameContainer.className = 'name'
        nameContainer.textContent = name
        return res
    }

    /**
     * @returns {HTMLElement}
     */
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
     * @param {MarkKey} key
     * @param {boolean} v
     */
    mark(key, v) {
        // For now all keys just map directly to a CSS class.
        if (v) {
            this.root.classList.add(key)
        } else {
            this.root.classList.remove(key)
        }
    }
}

/**
 * @typedef {'hovered'|'highlighted'|'matched'|'containsMatches'|'hasNoMatches'} MarkKey
 */

export class FileDom {
    /**
     * @param {File} file
     */
    constructor(file) {
        this.file = file
        this.root = FileDom.#createRoot(file.name)
        domMap.set(this.root, this)
    }

    /**
     * @param {string} name
     * @returns {HTMLElement}
     */
    static #createRoot(name) {
        const res = document.createElement('li')
        res.className = 'file'
        res.textContent = name
        return res
    }

    /**
     * @param {MarkKey} key
     * @param {boolean} v
     */
    mark(key, v) {
        // For now all keys just map directly to a CSS class.
        if (v) {
            this.root.classList.add(key)
        } else {
            this.root.classList.remove(key)
        }
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
     * @param {Controller} controller
     */
    constructor(target, controller) {
        this.target = target
        this.root = TargetContainerDom.#createRoot()
        this.container = this.root.appendChild(TargetContainerDom.#createContainer())
        controller.registerEventListeners(this)
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
     * @param {string} event
     * @param {EventListenerOrEventListenerObject} listener
     */
    addEventListener(event, listener) {
        this.root.addEventListener(event, listener)
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
