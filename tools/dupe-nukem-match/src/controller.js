import {Dir, File, Target, walkDir} from "./domain.js"
import {DirDom, domMap, FileDom, TargetContainerDom} from "./dom.js"

/** @typedef {import("./dom.js").DynamicMarkKey} DynamicMarkKey */
/**
 * @template Key
 * @typedef {import("./dom.js").Markable<Key>} Markable
 */

export class Controller {
    /**
     * @param {Target[]} targets
     */
    constructor(targets) {
        this.targets = targets

        /** @type {{[K in DynamicMarkKey]: Set<{dom: Markable<K>|null}>|null}} */
        this.marks = {
            'contains-matching': null,    // dir
            'contains-nonmatching': null, // dir
            'selected': null,             // dir or file
            'matching': null,             // file
        }

        /** @type {EventTarget|null} */
        this.deferredTarget = null
    }

    /**
     * @param {TargetContainerDom|Node} dom
     */
    registerEventListeners(dom) {
        dom.addEventListener('mouseover', e => {
            if (e instanceof MouseEvent && !e.altKey) {
                this.deferredTarget = e.target
                return
            }
            this.selectTarget(e.target)
        })
        dom.addEventListener('mouseout', e => {
            if (e instanceof MouseEvent && !e.altKey) {
                // Ignore event unless alt key is pressed.
                this.deferredTarget = null
                return
            }
            this.clearMarks()
        })
        document.addEventListener('keydown', ({altKey}) => {
            if (altKey) {
                this.selectTarget(this.deferredTarget)
            }
        })
    }

    /**
     * @param {Set<File>} files
     * @returns {Set<File>}
     */
    findMatchesOf(files) {
        /** @type {Set<File>} */
        let res = new Set()
        /** @type {Set<number>} */
        let hashes = new Set()
        for (let {hash} of files) {
            hashes.add(hash)
        }
        for (let hash of hashes) {
            for (let t of this.targets) {
                let matches = t.index.get(hash)
                if (matches !== undefined) for (let f of matches) {
                    if (!files.has(f)) {
                        res.add(f)
                    }
                }
            }
        }
        return res
    }

    /**
     * @template {keyof typeof this.marks} K
     * @param {K} key
     * @param {typeof this.marks[K]} nodes
     */
    refreshMarks(key, nodes) {
        let marked = this.marks[key]
        if (marked !== null) for (let node of marked) {
            if (!nodes?.has(node)) {
                node.dom?.mark(key, false)
            }
        }
        if (nodes !== null) for (let node of nodes) {
            if (!marked?.has(node)) {
                node.dom?.mark(key, true)
            }
        }
        this.marks[key] = nodes
    }

    clearMarks() {
        for (let key of Object.keys(this.marks)) {
            // Type annotation is necessary because 'Object.keys' returns 'string[]'.
            this.refreshMarks(/** @type {DynamicMarkKey} */ (key), null)
        }
    }

    /* EVENT HANDLING */

    /**
     * @param {EventTarget|null} target
     */
    selectTarget(target) {
        /** @type {Set<Dir|File>} */
        let selected = new Set()
        while (target !== null) {
            // As we only have a single event listener, we cannot rely on the event bubbling to the parent element
            // when we hit a DOM node sitting above the dir/file elements (like the 'name' div of a Dir).
            // Instead, we walk up the DOM tree manually until we find a hit.
            if (target instanceof HTMLElement) {
                let dom = domMap.get(target);
                if (dom instanceof FileDom) {
                    selected.add(dom.file)
                    break;
                }
                if (dom instanceof DirDom) {
                    selected.add(dom.dir)
                    // Collect all files and directories in the subtree for highlighting.
                    walkDir(
                        dom.dir,
                        (f) => selected.add(f),
                        (d, level) => {
                            // level > 0 && selected.add(d);
                            return true;
                        },
                    )
                    break;
                }
                // Target is not a domain node: bubble on...
                target = target.parentElement;
            }
        }
        this.refreshMarks('selected', selected)

        // Match against hovered and selected files.
        /** @type {Set<File>} */
        let filesToMatch = new Set()
        selected.forEach(s => s instanceof File && filesToMatch.add(s))
        let matchingFiles = this.findMatchesOf(filesToMatch);
        this.refreshMarks('matching', matchingFiles)

        // Collect all parent directories of any files that are matched.
        /** @type {Set<Dir>} */
        let dirsContainingMatchedFiles = new Set()
        matchingFiles.forEach(f => f.forEachAncestor(a => dirsContainingMatchedFiles.add(a)))
        this.refreshMarks('contains-matching', dirsContainingMatchedFiles)
    }
}