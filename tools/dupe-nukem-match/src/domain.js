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
    updateMatchInfo(otherTargets) {
        const ownTarget = this;
        /**
         * @param {Dir} dir
         * @return {Set<number>}
         */
        function updateDir(dir) {
            /** @type {Set<number>} */
            const hashes = new Set()
            for (const d of dir.dirs) {
                const dirHashes = updateDir(d)
                dirHashes.forEach(h => hashes.add(h))
            }
            for (const f of dir.files) {
                updateFile(f)
                hashes.add(f.hash)
            }
            dir.refreshMatchState(hashes)
            dir.syncDom()

            return hashes
        }

        /** @param {File} file */
        function updateFile(file) {
            // For now we're including matches in our own target - but it's unclear if (and how) we should...
            file.refreshMatchState(ownTarget, otherTargets)
            file.syncDom()
        }

        updateDir(this.root)
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
     * @param {Dir|undefined} parent Parent directory.
     * @param {string} name
     */
    constructor(parent, name) {
        this.parent = parent
        this.name = name
        this.dom = null // init deferred to create circular reference

        /* TREE NAVIGATION */

        /** @type {Dir[]} */
        this.dirs = []
        /** @type {File[]} */
        this.files = []

        /* MATCH STATE (operating on files rather than hashes for now) */

        /** @type {number} */
        this.totalFileCount = 0;
        /** @type {number} */
        this.matchedByOtherTargetCount = 0;
        /** @type {number} */
        this.matchedByOwnOrOtherTargetCount = 0;
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
     * @param {Set<number>} hashes
     */
    refreshMatchState(hashes) {
        // Collect dirs (
        // let totalFileCount = 0;
        // let matchedByOtherTargetCount = 0
        // let matchedByOwnOrOtherTargetCount = 0;
        // for (const d of this.dirs) {
        //     totalFileCount += d.totalFileCount
        //     matchedByOtherTargetCount += d.matchedByOtherTargetCount
        //     matchedByOwnOrOtherTargetCount += d.matchedByOwnOrOtherTargetCount
        // }
        // for (const f of this.files) {
        //     totalFileCount++
        //     if (f.matchedByOwnTarget || f.matchedByOtherTarget) {
        //         matchedByOwnOrOtherTargetCount++
        //     }
        //     if (f.matchedByOtherTarget) {
        //         matchedByOtherTargetCount++
        //     }
        // }
        // this.totalFileCount = totalFileCount
        // this.matchedByOtherTargetCount = matchedByOtherTargetCount
        // this.matchedByOwnOrOtherTargetCount = matchedByOwnOrOtherTargetCount
    }

    syncDom() {
        if (this.totalFileCount < this.matchedByOtherTargetCount) {
            this.dom?.mark('hasNoMatches', true)
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

        /* MATCH STATE */

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
        /** @type {Dir|undefined} */
        let d = this.dir
        let c = 0
        do {
            callback(d, c++)
            d = d.parent
        } while (d)
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

    // Is separate method because we might want to pass some settings,
    // allowing us to update DOM without recomputing state.
    syncDom() {
        if (!this.matchedByOwnTarget && !this.matchedByOtherTarget) {
            this.dom?.mark('hasNoMatches', true)
        }
    }
}

/**
 * Create new {@link Dir}, including the DOM element into which it will be rendered.
 * @param {Dir|undefined} parent
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
     * @param {Dir|undefined} parent
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

    const root = buildRecursive(scanRoot, undefined)
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
