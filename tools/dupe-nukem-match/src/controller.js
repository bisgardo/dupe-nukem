import {Target, walkDir} from "./domain.js";
import {DirDom, domMap, FileDom, TargetContainerDom} from "./dom.js";

/**
 * @typedef {import("./dom.js").MarkKey} MarkKey
 */

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
                const dom = domMap.get(target);
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
                // Target is not a "root" DOM node - let handler "bubble" up the tree.
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