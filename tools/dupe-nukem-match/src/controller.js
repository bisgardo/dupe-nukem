import {Dir, File, Target, walkDir} from "./domain.js"
import {DirDom, FileDom, TargetContainerDom, domMap} from "./dom.js"

/** @typedef {import("./domain.js").Hash} Hash */
/** @typedef {import("./domain.js").HashCounts} HashCounts */

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
            'contains-matching': null,      // dir
            'contains-all-matching': null,  // dir
            'selected': null,               // dir or file
            'matching': null,               // file
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
     * @param {Set<Hash>} hashes
     * @param {Set<Dir|File>} selected
     * @returns {Set<File>}
     */
    findMatchesOf(hashes, selected) {
        /** @type {Set<File>} */
        let res = new Set()
        for (let hash of hashes) {
            for (let t of this.targets) {
                let matches = t.index.get(hash)
                if (matches !== undefined) for (let f of matches) {
                    if (!selected.has(f)) {
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

        /**
         * @param {HTMLElement} el
         * @return {EventTarget|null} Next target to attempt.
         */
        function handle(el) {
            let dom = domMap.get(el)
            if (dom instanceof FileDom) {
                selected.add(dom.file)
                return null
            }
            if (dom instanceof DirDom) {
                selected.add(dom.dir)
                if (dom.dir.hashes === null) {
                    throw new Error(`hashes of dir '${dom.dir.name}' has not yet been initialized`)
                }
                // Collect all files and directories in the subtree for highlighting.
                walkDir(
                    dom.dir,
                    (f) => selected.add(f),
                    (d, level) => {
                        // level > 0 && selected.add(d)
                        return true
                    },
                )
                return null
            }
            return el.parentElement
        }

        while (target instanceof HTMLElement) {
            // As we only have a single event listener, we cannot rely on the event bubbling to the parent element
            // when we hit a DOM node sitting above the dir/file elements (like the 'name' div of a Dir).
            // Instead, we walk up the DOM tree manually until we find a hit.
            target = handle(target)
        }
        this.refreshMarks('selected', selected)
        this.refreshMatches(selected)
    }

    /**
     * @param {Set<Dir|File>} selected
     */
    refreshMatches(selected) {
        /** @type {Set<Hash>} */
        let hashes = new Set()
        for (let s of selected) {
            if (s instanceof File) hashes.add(s.hash)
        }
        let matchingFiles = this.findMatchesOf(hashes, selected)
        this.refreshMarks('matching', matchingFiles)

        // Collect all parent directories of any files that are matched.
        /** @type {Set<Dir>} */
        let dirsContainingMatchedFiles = new Set()
        matchingFiles.forEach(f => f.forEachAncestor(a => dirsContainingMatchedFiles.add(a)))
        this.refreshMarks('contains-matching', dirsContainingMatchedFiles)

        // /**
        //  * @param {Dir} d
        //  */
        // function containsOnlyMatchedFiles(d) {
        //     if (d.hashes === null) {
        //         throw new Error(`hashes of dir '${d.name}' have not yet been initialized`)
        //     }
        //     for (let h of d.hashes.keys()) {
        //         if (!hashes.has(h)) {
        //             return false
        //         }
        //     }
        //     return true
        // }

        /**
         * @param {Dir} d
         */
        function containsAllMatchedFiles(d) {
            if (d.hashes === null) {
                throw new Error(`hashes of dir '${d.name}' have not yet been initialized`)
            }
            // IDEA: Could count number of matched hashes for more stats...
            for (let h of hashes) {
               if (!d.hashes.has(h)) {
                   return false
               }
            }
            return true
        }

        let dirsContainingAllMatchedFiles = new Set()
        for (let d of dirsContainingMatchedFiles) {
            if (containsAllMatchedFiles(d)) dirsContainingAllMatchedFiles.add(d)
        }
        this.refreshMarks('contains-all-matching', dirsContainingAllMatchedFiles)
    }
}
