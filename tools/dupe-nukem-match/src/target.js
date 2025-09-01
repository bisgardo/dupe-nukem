/**
 * The main domain model of the app:
 * The app compares a list of {@link Target}s, each of which is loaded from a {@link ScanResult}.
 * A target contains a navigable tree of the file structure and a {@link FileIndex} of files by hash.
 * Each directory and file (represented by {@link Dir} and {@link File}, respectively) has a "DOM component manager"
 * which is responsible for directly manipulating the DOM elements that render the target.
 */

import {assertScanDir, assertScanFile} from './scan'
import {DirDom, FileDom} from './dom'

// Import JSDoc types. This could be done once and for all in a globals.d.ts file.
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
class Dir {
    /**
     * @param {Dir|undefined} parent Parent directory.
     * @param {ScanDir} scanDir Source directory from the scan result.
     * @param {DirDom} dom DOM manager of the directory.
     */
    constructor(parent, scanDir, dom) {
        this.parent = parent
        this.scanDir = scanDir
        this.dom = dom
    }
}

/**
 * A file within a directory structure, including its associated DOM element.
 */
class File {
    /**
     * @param {Dir} dir Directory in which the file is located.
     * @param {ScanFile} scanFile Source file from the scan result.
     * @param {FileDom} dom DOM manager of the file.
     */
    constructor(dir, scanFile, dom) {
        this.dir = dir
        this.scanFile = scanFile
        this.dom = dom
    }
}

/**
 * Create new {@link Dir}, including the DOM element into which it will be rendered.
 * @param {Dir|undefined} parent
 * @param {ScanDir} scanDir
 * @returns Dir
 */
function makeTargetDir(parent, scanDir) {
    const dom = new DirDom(scanDir.name)
    parent?.dom.append(dom) // attach to parent's container DOM element (if provided)
    return new Dir(parent, scanDir, dom)
}

/**
 * Create new {@link File}, including the DOM element into which it will be rendered.
 * @param {Dir} dir
 * @param {ScanFile} scanFile
 * @returns File
 */
function makeTargetFile(dir, scanFile) {
    const dom = new FileDom(scanFile.name)
    dir.dom.append(dom) // attach to parent's container DOM element
    return new File(dir, scanFile, dom)
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
    return {root, index}
}
