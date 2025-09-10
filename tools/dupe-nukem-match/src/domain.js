/**
 * The main domain model of the app:
 * The app compares a list of {@link Target}s, each of which is loaded from a {@link ScanResult}.
 * A target contains a navigable tree of the file structure and a {@link FileIndex} of files by hash.
 * Each directory and file (represented by {@link Dir} and {@link File}, respectively) has a "DOM component manager"
 * which is responsible for directly manipulating the DOM elements that render the target.
 */

import {assertScanDir, assertScanFile} from './scan'
import {DirDom, FileDom} from './dom'

/**
 * All processed information of a root.
 */
export class Target {
    /**
     * @param {Dir} root
     * @param {FileIndex} index
     */
    constructor(root, index) {
        this.root = root
        this.index = index
    }

    /**
     * @param {Target[]} otherTargets
     */
    refreshMatchState(otherTargets) {
        /**
         * @param {Dir} dir
         */
        const visitDir = (dir) => {
            /** @type {Set<number>} */
            for (const d of dir.dirs) {
                visitDir(d)
            }
            for (const f of dir.files) {
                visitFile(f)
            }
            dir.refreshMatchState(this, otherTargets)
        }
        /** @param {File} file */
        const visitFile = (file) => {
            file.refreshMatchState(this, otherTargets)
        }
        visitDir(this.root)
    }

    syncDom() {
        /**
         * @param {Dir} dir
         */
        const visitDir = (dir) => {
            /** @type {Set<number>} */
            for (const d of dir.dirs) {
                visitDir(d)
            }
            for (const f of dir.files) {
                visitFile(f)
            }
            dir.syncDom()
        }
        /** @param {File} file */
        const visitFile = (file) => {
            file.syncDom()
        }
        visitDir(this.root)
    }
}

/**
 * Index of all files in a target.
 * @typedef {Map<number, File[]>} FileIndex
 */

/**
 * A directory in a hierarchical file structure, including its associated DOM elements.
 */
export class Dir {
    /**
     * @param {Dir|null} parent Parent directory.
     * @param {string} name
     */
    constructor(parent, name) {
        this.parent = parent
        this.name = name
        this.dom = null // init deferred to create circular reference

        /* TREE NAVIGATION - populated while subtree is being constructed */

        /** @type {Dir[]} */
        this.dirs = []
        /** @type {File[]} */
        this.files = []

        /* MATCH STATE - populated in 'refreshMatchState' */
        /** @type {number} */
        this.totalFileCount = 0
        /** @type {Set<number>|null} */
        this.hashes = null
        /** @type {Set<Dir>|null} */
        this.dirsInOwnTargetContainingMatches = null
        /** @type {Set<Dir>[]|null} */
        this.dirsInOtherTargetsContainingMatches = null
    }

    /**
     * Register the DOM manager of this directory and optionally attach it to the parent dir's DOM (if there is one).
     * @param {DirDom} dom DOM manager of the directory.
     * @param {boolean} attach
     */
    initDom(dom, attach) {
        this.dom = dom
        if (attach && this.parent?.dom) {
            dom.appendTo(this.parent.dom)
        }
    }

    /**
     * @param {Dir} child
     */
    addDir(child) {
        this.dirs.push(child)
    }

    /**
     * @param {File} child
     */
    addFile(child) {
        this.files.push(child)
    }

    /**
     * @param {Target} ownTarget
     * @param {Target[]} otherTargets
     */
    refreshMatchState(ownTarget, otherTargets) {
        // Collect hashes of subtree (and total file count).
        let totalFileCount = 0
        const hashes = new Set()
        for (const d of this.dirs) {
            if (d.hashes === null) {
                throw new TypeError(`field 'hashes' of Dir '${d}' has not been initialized`)
            }
            totalFileCount += d.totalFileCount
            d.hashes.forEach(h => hashes.add(h))
        }
        for (const f of this.files) {
            totalFileCount++
            hashes.add(f.hash)
        }
        this.totalFileCount = totalFileCount
        this.hashes = hashes

        /**
         * @param {Target} target
         * @param {Set<number>} hashes
         * @return {Set<Dir>}
         */
        const collectMatchDirs = (target, hashes) => {
            /** @type {Set<Dir>} */
            const res = new Set();
            for (const h of hashes) {
                target.index.get(h)?.forEach(f => res.add(f.dir))
            }
            // Remove descendants of matches as well as the dir's own subtree.
            res.delete(this)
            for (const d of res) {
                let p = d.parent
                while (p !== null) {
                    if (p === this || res.has(p)) {
                        res.delete(d)
                        break
                    }
                    p = p.parent
                }
            }
            return res
        }

        this.dirsInOwnTargetContainingMatches = collectMatchDirs(ownTarget, hashes)
        this.dirsInOtherTargetsContainingMatches = otherTargets.map(t => collectMatchDirs(t, hashes))
        // Interesting question is now which of these dirs contain *all* hashes!
        // Note that any common shared ancestor could contain all matches without any of the dirs doing so individually.
    }

    /**
     * @param {boolean} checkOwnTarget
     * @param {boolean} checkOtherTargets
     * @return {Set<number>}
     */
    unmatchedHashes(checkOwnTarget, checkOtherTargets) {
        /** @type {Set<number>} */
        const res = new Set(this.hashes)
        if (checkOwnTarget) {
            if (this.dirsInOwnTargetContainingMatches === null) {
                throw new TypeError(`field 'dirsInOwnTargetContainingMatches' of Dir '${this}' has not been initialized`)
            }
            for (const d of this.dirsInOwnTargetContainingMatches) {
                if (d.hashes === null) {
                    throw new TypeError(`field 'hashes' of Dir '${d}' has not been initialized`)
                }
                for (const h of d.hashes) {
                    res.delete(h)
                }
            }
        }
        if (checkOtherTargets) {
            if (this.dirsInOtherTargetsContainingMatches === null) {
                throw new TypeError(`field 'dirsInOtherTargetsContainingMatches' of Dir '${this}' has not been initialized`)
            }
            for (const dirsContainingMatches of this.dirsInOtherTargetsContainingMatches) {
                for (const d of dirsContainingMatches) {
                    if (d.hashes === null) {
                        throw new TypeError(`field 'hashes' of Dir '${d}' has not been initialized`)
                    }
                    for (const h of d.hashes) {
                        res.delete(h)
                    }
                }
            }
        }
        return res
    }

    syncDom() {
        if (this.hashes === null) {
            throw new TypeError(`field 'hashes' of Dir '${this}' has not been initialized`)
        }
        if (this.dom === null) {
            throw new TypeError(`field 'dom' of Dir '${this}' has not been initialized`)
        }
        // Why all this crap instead of just ask all files recursively for their state, you ask?
        // Well, file could be matched internally in the target.
        // This means that the file is matched up to some ancestor directory, but not outside of it.

        // TODO: Instead of storing hashes as a set, make it a map from hash to match count.
        //       Then you can check with the number of matches in the target index to see if there are more matches that what's inside the tree!
        //       Use this to determine right away
        //       * whether any file (hash) within the tree has matches outside the tree
        //       * whether any file (hash) within the tree is not matched outside the tree
        //       there are any matches within the target but outside the dir's own tree.
        //       Only in a later pass do we check against other targets (which is trivial).
        if (this.unmatchedHashes(true, true).size > 0) {
            this.dom.mark('containsUnmatched', true)
        }
    }
}

/**
 * A file within a directory structure, including its associated DOM element.
 */
export class File {
    /**
     * @param {Dir} dir Directory in which the file is located.
     * @param {string} name
     * @param {number} size
     * @param {number} hash
     */
    constructor(dir, name, size, hash) {
        this.dir = dir
        this.name = name
        this.size = size
        this.hash = hash
        this.dom = null // init deferred to create circular reference

        /* MATCH STATE - populated in 'refreshMatchState' */

        /** @type {boolean|null} */
        this.matchedByOwnTarget = null
        /** @type {boolean|null} */
        this.matchedByOtherTarget = null
    }

    /**
     * Register the DOM manager of this directory and optionally attach it to the containing dir's DOM.
     * @param {FileDom} dom DOM manager of the file.
     * @param {boolean} attach
     */
    initDom(dom, attach) {
        this.dom = dom
        if (attach && this.dir.dom) {
            dom.appendTo(this.dir.dom)
        }
    }

    /**
     * @param {(value: Dir, index: number) => void} callback
     */
    forEachAncestor(callback) {
        /** @type {Dir|null} */
        let d = this.dir
        let c = 0
        do {
            callback(d, c++)
            d = d.parent
        } while (d !== null)
    }

    /**
     * Set the match state of this file and sync it to the DOM.
     * @param {Target} ownTarget
     * @param {Target[]} otherTargets
     */
    refreshMatchState(ownTarget, otherTargets) {
        const numMatchesOwnTarget = ownTarget.index.get(this.hash)?.length ?? 0;
        this.matchedByOwnTarget = numMatchesOwnTarget > 1
        this.matchedByOtherTarget = otherTargets.some((target) => {
            const numMatches = target.index.get(this.hash)?.length ?? 0
            return numMatches > 0
        })
    }

    syncDom() {
        if (this.dom === null) {
            throw new TypeError(`field 'dom' of File '${this}' has not been initialized`)
        }
        // Is separate method because we might want to pass some settings,
        // allowing us to update DOM without recomputing state.
        if (!this.matchedByOwnTarget && !this.matchedByOtherTarget) {
            this.dom.mark('unmatched', true)
        }
    }
}

/**
 * Create new {@link Dir}, including the DOM element into which it will be rendered.
 * @param {Dir|null} parent
 * @param {string} name
 * @returns Dir
 */
function makeTargetDir(parent, name) {
    const res = new Dir(parent, name)
    parent?.addDir(res)
    const dom = new DirDom(res)
    res.initDom(dom, true)
    return res
}

/**
 * Create new {@link File}, including the DOM element into which it will be rendered.
 * @param {Dir} dir
 * @param {string} name
 * @param {number} size
 * @param {number} hash
 * @returns File
 */
function makeTargetFile(dir, name, size, hash) {
    const res = new File(dir, name, size, hash)
    dir.addFile(res)
    const dom = new FileDom(res)
    res.initDom(dom, true)
    return res
}

/**
 * Build target from the root of a {@link ScanResult}.
 * @param {unknown} scanRoot
 * @returns {Target}
 */
export function buildTarget(scanRoot) {
    /** @type {FileIndex} */
    const index = new Map()

    /**
     * @param {unknown} scanDir
     * @param {Dir|null} parent
     */
    function buildRecursive(scanDir, parent) {
        assertScanDir(scanDir)
        const targetDir = makeTargetDir(parent, scanDir.name)
        if (scanDir.dirs !== undefined) for (const d of scanDir.dirs) {
            buildRecursive(d, targetDir)
        }
        if (scanDir.files !== undefined) for (const f of scanDir.files) {
            assertScanFile(f)
            const matchFile = makeTargetFile(targetDir, f.name, f.size, f.hash)
            const matchFiles = index.get(f.hash)
            if (matchFiles === undefined) {
                index.set(f.hash, [matchFile])
            } else {
                matchFiles.push(matchFile)
            }
        }
        return targetDir
    }

    const root = buildRecursive(scanRoot, null)
    return new Target(root, index)
}

/**
 * @param {Dir} dir
 * @param {(file: File, level: number) => void} fileCallback
 * @param {(dir: Dir, level: number) => boolean} dirCallback
 * @param level
 */
export function walkDir(dir, fileCallback, dirCallback, level = 0) {
    if (dirCallback(dir, level++)) {
        for (const d of dir.dirs) {
            walkDir(d, fileCallback, dirCallback, level)
        }
        for (const f of dir.files) {
            fileCallback(f, level)
        }
    }
}
