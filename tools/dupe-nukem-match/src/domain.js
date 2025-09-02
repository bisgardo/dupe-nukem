/**
 * The main domain model of the app:
 * The app compares a list of {@link Target}s, each of which is loaded from a {@link ScanResult}.
 * A target contains a navigable tree of the file structure and a {@link FileIndex} of files by hash.
 * Each directory and file (represented by {@link Dir} and {@link File}, respectively) has a "DOM component manager"
 * which is responsible for directly manipulating the DOM elements that render the target.
 */

import {assertScanDir, assertScanFile} from './scan'
import {DirDom, FileDom} from './dom'

// Import JSDoc types. This could be done once and for all in 'globals.d.ts'.
// But it feels like we're probably going to move the fields to the target types
// (and make them fully navigable) rather than keeping these "raw" input values around.
/** @typedef {import("./scan.js").ScanDir} ScanDir */
/** @typedef {import("./scan.js").ScanFile} ScanFile */

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
     * @param {Target[]} targets
     */
    updateMatchInfo(targets) {
        const ownTargetIdx = targets.indexOf(this)

        /** @param {Dir} dir */
        function updateDir(dir) {
            for (const d of dir.dirs) updateDir(d)
            for (const f of dir.files) updateFile(f)
            // TODO: update dir information
        }

        /** @param {File} file */
        function updateFile(file) {
            const {hash} = file.scanFile;
            // For now we're including matches in our own target - but it's unclear if (and how) we should...
            file.setMatchState(
                targets.map((target, idx) => {
                    let numMatches = target.index.get(hash)?.length ?? 0;
                    if (idx === ownTargetIdx) numMatches-- // subtract self
                    console.assert(numMatches >= 0)
                    return numMatches;
                })
            )
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
     * @param {ScanDir} scanDir Source directory from the scan result.
     */
    constructor(parent, scanDir) {
        this.parent = parent
        this.scanDir = scanDir
        this.dom = null; // init deferred to create circular reference

        /* TREE NAVIGATION */

        /** @type {Dir[]} */
        this.dirs = []
        /** @type {File[]} */
        this.files = []

        /* MATCH STATE */
        // TODO
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
}

/**
 * A file within a directory structure, including its associated DOM element.
 */
export class File {
    /**
     * @param {Dir} dir Directory in which the file is located.
     * @param {ScanFile} scanFile Source file from the scan result.
     */
    constructor(dir, scanFile) {
        this.dir = dir
        this.scanFile = scanFile
        this.dom = null; // init deferred to create circular reference

        /* MATCH STATE */

        /** @type {number[]|null} */
        this.matchCountByTarget = null;
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
     * @param {number[]} matchCountByTarget Number of matches in other targets.
     */
    setMatchState(matchCountByTarget) {
        this.matchCountByTarget = matchCountByTarget

        // Sync DOM.
        const totalMatchCount = matchCountByTarget.reduce((acc, c) => acc+c, 0);
        if (totalMatchCount === 0) {
            this.dom?.mark('hasNoMatches', true)
        }
    }
}

/**
 * Create new {@link Dir}, including the DOM element into which it will be rendered.
 * @param {Dir|undefined} parent
 * @param {ScanDir} scanDir
 * @returns Dir
 */
function makeTargetDir(parent, scanDir) {
    const res = new Dir(parent, scanDir)
    parent?.addDir(res)
    const dom = new DirDom(res)
    res.initDom(dom, true)
    return res
}

/**
 * Create new {@link File}, including the DOM element into which it will be rendered.
 * @param {Dir} dir
 * @param {ScanFile} scanFile
 * @returns File
 */
function makeTargetFile(dir, scanFile) {
    const res = new File(dir, scanFile)
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
        const targetDir = makeTargetDir(parent, scanDir)
        if (scanDir.dirs !== undefined) {
            for (const d of scanDir.dirs) {
                buildRecursive(d, targetDir)
            }
        }
        if (scanDir.files !== undefined) {
            for (const f of scanDir.files) {
                assertScanFile(f)
                const matchFile = makeTargetFile(targetDir, f)
                const matchFiles = index.get(f.hash)
                if (matchFiles === undefined) {
                    index.set(f.hash, [matchFile])
                } else {
                    matchFiles.push(matchFile)
                }
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
