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

import {Dir, File, Target, walkDir} from "./target"

/** @type {WeakMap<HTMLElement, DirDom|FileDom>} */
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

    // TODO: Add method for expanding, collapsing etc.
}

/**
 * @typedef {'hovered'|'highlighted'|'matched'|'containsMatches'} MarkKey
 */

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
     * @template {keyof HTMLElementEventMap} K
     * @param {K} event
     * @param {(this: HTMLDivElement, ev: HTMLElementEventMap[K]) => any} listener
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

export class Controller {
    /**
     * @param {Target[]} targets
     */
    constructor(targets) {
        this.targets = targets

        /** @type {Record<MarkKey, Set<DirDom|FileDom>|null>} */
        this.marks = {
            hovered: null,
            highlighted: null,
            matched: null,
            containsMatches: null,
        }
    }

    /**
     * @param {TargetContainerDom} dom
     */
    registerEventListeners(dom) {
        dom.addEventListener('mouseover', this.handleMouseOver)
        dom.addEventListener('mouseout', this.handleMouseOut)
    }

    /**
     * @param {Set<FileDom>} doms
     * @returns {Set<FileDom>}
     */
    findMatchesOf(doms) {
        /** @type {Set<FileDom>} */
        const res = new Set()
        /** @type {Set<number>} */
        const hashes = new Set()
        for (const {file} of doms) {
            hashes.add(file.scanFile.hash)
        }
        for (const hash of hashes) {
            for (const t of this.targets) {
                const matches = t.index.get(hash)
                if (matches !== undefined) for (const {dom} of matches) {
                    if (dom && !doms.has(dom)) {
                        res.add(dom)
                    }
                }
            }
        }
        return res
    }

    /**
     * @param {MarkKey} key
     * @param {Set<DirDom|FileDom>|null} doms
     */
    refreshMarks(key, doms) {
        const marked = this.marks[key];
        if (marked !== null) for (const dom of marked) {
            if (!doms?.has(dom)) {
                dom.mark(key, false)
            }
        }
        if (doms !== null) for (const dom of doms) {
            if (!marked?.has(dom)) {
                dom.mark(key, true)
            }
        }
        this.marks[key] = doms
    }

    clearMarks() {
        for (const key of Object.keys(this.marks)) {
            // Type annotation is necessary because 'Object.keys' returns 'string[]'.
            this.refreshMarks(/** @type {MarkKey} */ (key), null)
        }
    }

    /**
     * @param {MouseEvent} e
     */
    handleMouseOver = ({target}) => {
        /** @type {Set<DirDom|FileDom>} */
        const hovered = new Set();
        /** @type {Set<DirDom|FileDom>} */
        const highlighted = new Set()
        while (target !== null) {
            // As we only have a single event listener, we cannot rely on the event bubbling to the parent element
            // when we hit a DOM node sitting above the dir/file elements (like the 'name' div of a Dir).
            // Instead, we walk up the DOM tree manually until we find a hit (at which point we 'break' out of the loop).
            if (target instanceof HTMLElement) {
                const dom = elements.get(target);
                if (dom instanceof FileDom) {
                    hovered.add(dom)
                    // highlighted.add(dom)
                    break;
                }
                if (dom instanceof DirDom) {
                    hovered.add(dom)
                    // Collect all files and directories in the subtree for highlighting.
                    walkDir(
                        dom.dir,
                        ({dom}) => dom && highlighted.add(dom),
                        ({dom}, level) => {
                            // level > 0 && dom && highlighted.add(dom);
                            return true;
                        },
                    )
                    break;
                }
                target = target.parentElement;
            }
        }
        // Array.of(hovered).map(d => d.mark('hovered', true))
        this.refreshMarks('hovered', hovered)
        this.refreshMarks('highlighted', highlighted)

        // Match against hovered and highlighted files.
        /** @type {Set<FileDom>} */
        const filesToMatch = new Set()
        hovered.forEach(d => d instanceof FileDom && filesToMatch.add(d))
        highlighted.forEach(d => d instanceof FileDom && filesToMatch.add(d))
        const matchingFiles = this.findMatchesOf(filesToMatch);
        this.refreshMarks('matched', matchingFiles)

        // Collect all parent directories of any files that are matched.
        /** @type {Set<DirDom>} */
        const dirsContainingMatchedFiles = new Set()
        matchingFiles.forEach(({file}) => file.ancestors.forEach(({dom}) => dom && dirsContainingMatchedFiles.add(dom)))
        this.refreshMarks('containsMatches', dirsContainingMatchedFiles)
    }

    handleMouseOut = () => {
        this.clearMarks()
    }
}
