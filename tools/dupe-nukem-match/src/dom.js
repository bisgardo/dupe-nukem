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

// Terminology:
//   Static (don't depend on selected):
//   - file is matched in own target : matched
//   - file is matched in other target : matched
//   - directory contains some files that are not matched in own target : contains-unmatched
//   - directory contains some files that are not matched in other target : contains-unmatched
//   - directory doesn't contain any files that are matched in own target : contains-no-matched
//   - directory doesn't contain any files that are matched in other target : contains-no-matched
//   Dynamic (depend on selected):
//   - file is selected : selected (currently: "hovered"/"highlighted")
//   - dir is selected : selected
//   - file matches selected file : matching
//   - directory contains files matching selected files : contains-matching
//   - directory contains files *not* matching selected files : contains-nonmatching
//   Idea: Render "static" property with text styling, "dynamic" ones with background/border?


import {Controller} from "./controller.js"
import {Dir, File, Target} from "./domain.js"

/** @typedef {'contains-unmatched'|'contains-no-matched'} DirStaticMarkKey */
/** @typedef {'matched-by-own-target'|'matched-by-other-target'} FileStaticMarkKey */
/** @typedef {'selected'|'contains-matching'|'contains-all-matching'} DirDynamicMarkKey */
/** @typedef {'selected'|'matching'} FileDynamicMarkKey */

/** @typedef {DirDynamicMarkKey|FileDynamicMarkKey} DynamicMarkKey */
/** @typedef {DirStaticMarkKey|FileStaticMarkKey} StaticMarkKey */


/** @type {Record<DynamicMarkKey|StaticMarkKey, string>} */
let markCssClass = {
    'selected': 'selected',                               // dir/file: dynamic
    'contains-unmatched': 'contains-unmatched',           // dir:      static
    'contains-no-matched': 'contains-no-matched',         // dir:      static
    'contains-matching': 'contains-matching',             // dir:      dynamic
    'contains-all-matching': 'contains-all-matching',     // dir:      dynamic
    'matched-by-own-target': 'matched-by-own-target',     // file:     static
    'matched-by-other-target': 'matched-by-other-target', // file:     static
    'matching': 'matching',                               // file:     dynamic
}

/** @type {WeakMap<HTMLElement, DirDom|FileDom>} */
export let domMap = new WeakMap()

/**
 * @template Key
 * @typedef {object} Markable
 * @property {(k: Key, v: boolean) => void} mark
 */

/**
 * @implements {Markable<DirStaticMarkKey|DirDynamicMarkKey>}
 */
export class DirDom {
    /**
     * @param {Dir} dir
     */
    constructor(dir) {
        this.dir = dir
        this.root = DirDom.#createRoot()
        let nameContainer = this.root.appendChild(DirDom.#createNameContainer(dir.name))
        this.container = this.root.appendChild(DirDom.#createContainer())
        // Make "select" affect labels only, and, in particular, not the spaces between files.
        domMap.set(nameContainer, this)
    }

    /**
     * @returns {HTMLElement}
     */
    static #createRoot() {
        let res = document.createElement('li')
        res.className = 'dir'
        return res
    }

    /**
     * @param {string} name
     * @returns {HTMLElement}
     */
    static #createNameContainer(name) {
        let res = document.createElement('div');
        res.className = 'name'
        res.textContent = name
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
     * @param {DirStaticMarkKey|DirDynamicMarkKey} key
     * @param {boolean} v
     */
    mark(key, v) {
        // For now all keys just map directly to a CSS class.
        let cssClass = markCssClass[key]
        if (v) {
            this.root.classList.add(cssClass)
        } else {
            this.root.classList.remove(cssClass)
        }
    }
}

/**
 * @implements {Markable<FileStaticMarkKey|FileDynamicMarkKey>}
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
        let res = document.createElement('li')
        res.className = 'file'
        res.textContent = name
        return res
    }

    /**
     * @param {FileStaticMarkKey|FileDynamicMarkKey} key
     * @param {boolean} v
     */
    mark(key, v) {
        // For now all keys just map directly to a CSS class.
        let cssClass = markCssClass[key]
        if (v) {
            this.root.classList.add(cssClass)
        } else {
            this.root.classList.remove(cssClass)
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
        let root = document.createElement('div')
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
