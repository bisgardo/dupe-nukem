import {Dir, File, Target, walkDir} from "./domain.js";
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

        /** @type {Record<MarkKey, Set<Dir|File>|null>} */
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
     * @param {Set<File>} files
     * @returns {Set<File>}
     */
    findMatchesOf(files) {
        /** @type {Set<File>} */
        const res = new Set()
        /** @type {Set<number>} */
        const hashes = new Set()
        for (const f of files) {
            hashes.add(f.scanFile.hash)
        }
        for (const hash of hashes) {
            for (const t of this.targets) {
                const matches = t.index.get(hash)
                if (matches !== undefined) for (const f of matches) {
                    if (!files.has(f)) {
                        res.add(f)
                    }
                }
            }
        }
        return res
    }

    /**
     * @param {MarkKey} key
     * @param {Set<Dir|File>|null} nodes
     */
    refreshMarks(key, nodes) {
        const marked = this.marks[key];
        if (marked !== null) for (const node of marked) {
            if (!nodes?.has(node)) {
                node.dom?.mark(key, false)
            }
        }
        if (nodes !== null) for (const node of nodes) {
            if (!marked?.has(node)) {
                node.dom?.mark(key, true)
            }
        }
        this.marks[key] = nodes
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
        /** @type {Set<Dir|File>} */
        const hovered = new Set();
        /** @type {Set<Dir|File>} */
        const highlighted = new Set()
        while (target !== null) {
            // As we only have a single event listener, we cannot rely on the event bubbling to the parent element
            // when we hit a DOM node sitting above the dir/file elements (like the 'name' div of a Dir).
            // Instead, we walk up the DOM tree manually until we find a hit (at which point we 'break' out of the loop).
            if (target instanceof HTMLElement) {
                const dom = domMap.get(target);
                if (dom instanceof FileDom) {
                    hovered.add(dom.file)
                    // highlighted.add(dom)
                    break;
                }
                if (dom instanceof DirDom) {
                    hovered.add(dom.dir)
                    // Collect all files and directories in the subtree for highlighting.
                    walkDir(
                        dom.dir,
                        (f) => highlighted.add(f),
                        (d, level) => {
                            // level > 0 && highlighted.add(d);
                            return true;
                        },
                    )
                    break;
                }
                // Target is not a "root" DOM node - let handler "bubble" up the tree.
                target = target.parentElement;
            }
        }
        this.refreshMarks('hovered', hovered)
        this.refreshMarks('highlighted', highlighted)

        // Match against hovered and highlighted files.
        /** @type {Set<File>} */
        const filesToMatch = new Set()
        hovered.forEach(d => d instanceof File && filesToMatch.add(d))
        highlighted.forEach(d => d instanceof File && filesToMatch.add(d))
        const matchingFiles = this.findMatchesOf(filesToMatch);
        this.refreshMarks('matched', matchingFiles)

        // Collect all parent directories of any files that are matched.
        /** @type {Set<Dir>} */
        const dirsContainingMatchedFiles = new Set()
        matchingFiles.forEach(f => f.ancestors.forEach(a => dirsContainingMatchedFiles.add(a)))
        this.refreshMarks('containsMatches', dirsContainingMatchedFiles)
    }

    handleMouseOut = () => {
        this.clearMarks()
    }

    /**
     * Initialize the DOM nodes to display static information such as whether they have any matches in any other target.
     */
    init() {
        for (const target of this.targets) {
            const matchedHashes = target.hashesInOtherTargets(this.targets)
            for (const [hash, files] of target.index) {
                if (!matchedHashes.has(hash)) {
                    for (const file of files) {
                        file.dom?.markHasNoMatches()
                    }
                }
            }
        }
    }
}