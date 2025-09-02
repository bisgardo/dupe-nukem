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
        /** @type {Dir[]} */
        this.dirs = []
        /** @type {File[]} */
        this.files = []
        this.dom = null
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
     * Register the DOM manager of this directory and optionally attach it to the parent dir's DOM (if there is one).
     * @param {DirDom} dom DOM manager of the directory.
     * @param {boolean} attach
     */
    setDom(dom, attach) {
        this.dom = dom
        if (attach && this.parent?.dom) {
            dom.appendTo(this.parent.dom)
        }
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
        this.dom = null
    }

    /**
     * Register the DOM manager of this directory and optionally attach it to the containing dir's DOM.
     * @param {FileDom} dom DOM manager of the file.
     * @param {boolean} attach
     */
    setDom(dom, attach) {
        this.dom = dom
        if (attach && this.dir.dom) {
            dom.appendTo(this.dir.dom)
        }
    }

    get ancestors() {
        /** @type {Dir[]} */
        const res = []
        /** @type {Dir|undefined} */
        let d = this.dir
        while (d) {
            res.push(d)
            d = d.parent
        }
        return res
    }

    /**
     * @param {Target} target
     */
    hasMatchesIn(target) {
        return target.index.has(this.scanFile.hash)
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
    res.setDom(dom, true)
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
    res.setDom(dom, true)
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
