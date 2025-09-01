// TODO:
//  - Should probably create custom components for Dir and File. Seems better than relying on plain ul/li.
//    We actually don't need to make it a webcomponent yet - that upgrade can always be done later!

import './style.css'

// Load scan result files.
import testResult1 from '../gendata/test1.json'
import testResult2 from '../gendata/test2.json'

/* TYPE DEFINITIONS */

/* = JSON input types = */

/**
 * The result of scanning the root directory.
 * @typedef {Object} ScanResult
 * @property {number} schema_version Schema version.
 * @property {unknown} root Root directory.
 */
/**
 * The result of scanning a directory.
 * @typedef {Object} ScanDir
 * @property {string} name
 * @property {unknown[]=} dirs
 * @property {unknown[]=} files
 * @property {string[]=} empty_files
 * @property {string[]=} skipped_files
 * @property {string[]=} skipped_dirs
 */
/**
 * The result of scanning a file.
 * @typedef {Object} ScanFile
 * @property {string} name
 * @property {number} size
 * @property {number} ts
 * @property {number} hash
 */

/** @type {unknown[]} */
const scanRoots = [testResult1.root, testResult2.root]

/**
 * All processed information of a root.
 */
class Target {
    /**
     * @param {Dir} root
     * @param {Index} index
     */
    constructor(root, index) {
        this.root = root
        this.index = index
    }
}

/**
 * Index of a target.
 * @typedef {Map<number, File[]>} Index
 */

/**
 * A directory in a hierarchical file structure, including its associated DOM elements.
 */
class Dir {
    /**
     * @param {Dir|undefined} parent Parent directory.
     * @param {ScanDir} scanDir Source directory from the scan result.
     * @param {HTMLLIElement} dom DOM element of the directory.
     * @param {HTMLUListElement} contentsDom DOM element of the directory's contents.'
     */
    constructor(parent, scanDir, dom, contentsDom) {
        this.parent = parent
        this.scanDir = scanDir
        this.dom = dom
        this.containerDom = contentsDom
    }
}

/**
 * A file within a directory structure, including its associated DOM element.
 */
class File {
    /**
     * @param {Dir} dir Directory in which the file is located.
     * @param {ScanFile} scanFile Source file from the scan result.
     * @param {HTMLLIElement} dom DOM element of the file.
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
    const dom = document.createElement('li')
    dom.textContent = scanDir.name
    parent?.containerDom.appendChild(dom) // attach to parent's container DOM element
    const containerDom = document.createElement('ul')
    dom.appendChild(containerDom) // attach to "own" DOM element
    return new Dir(parent, scanDir, dom, containerDom)
}

/**
 * Create new {@link File}, including the DOM element into which it will be rendered.
 * @param {Dir} dir
 * @param {ScanFile} scanFile
 * @returns File
 */
function makeTargetFile(dir, scanFile) {
    const dom = document.createElement('li')
    dom.textContent = scanFile.name
    dir.containerDom.appendChild(dom) // attach to parent's container DOM element
    return new File(dir, scanFile, dom)
}

/* = Runtime validation helpers = */

/**
 * Determine if value is a non-null object (record).
 * @param {unknown} v
 * @returns {v is Record<string, unknown>}
 */
function isRecord(v) {
    return typeof v === 'object' && v !== null
}

/**
 * Check if value is an array of strings.
 * @param {unknown} a
 * @returns {a is string[]}
 */
function isStringArray(a) {
    return Array.isArray(a) && a.every(x => typeof x === 'string')
}

/**
 * Assert that provided value is a valid {@link ScanFile}, otherwise throw.
 * @param {unknown} scanFile
 * @returns {asserts scanFile is ScanFile}
 */
function assertScanFile(scanFile) {
    if (!isRecord(scanFile)) throw new TypeError('Invalid ScanFile: not an object')
    const { name, size, ts, hash } = scanFile
    if (typeof name !== 'string') throw new TypeError('Invalid ScanFile.name')
    if (!Number.isFinite(size)) throw new TypeError('Invalid ScanFile.size')
    if (!Number.isFinite(ts)) throw new TypeError('Invalid ScanFile.ts')
    if (!Number.isFinite(hash)) throw new TypeError('Invalid ScanFile.hash')
}

/**
 * Assert (non-recursively) that provided value is a valid {@link ScanDir}, otherwise throw.
 * @param {unknown} scanDir
 * @returns {asserts scanDir is ScanDir}
 */
function assertScanDir(scanDir) {
    if (!isRecord(scanDir)) throw new TypeError('Invalid ScanDir: not an object')
    const { name, dirs, files, empty_files, skipped_files, skipped_dirs } = scanDir
    if (typeof name !== 'string') throw new TypeError('Invalid ScanDir.name')
    if (dirs !== undefined && !Array.isArray(dirs)) throw new TypeError('Invalid ScanDir.dirs')
    if (files !== undefined && !Array.isArray(files)) throw new TypeError('Invalid ScanDir.files')
    if (empty_files !== undefined && !isStringArray(empty_files)) throw new TypeError('Invalid ScanDir.empty_files')
    if (skipped_files !== undefined && !isStringArray(skipped_files)) throw new TypeError('Invalid ScanDir.skipped_files')
    if (skipped_dirs !== undefined && !isStringArray(skipped_dirs)) throw new TypeError('Invalid ScanDir.skipped_dirs')
}

/**
 * Build target from a scan result.
 * @param {unknown} scanRoot
 * @returns {Target}
 */
function buildTarget(scanRoot) {
    /** @type {Index} */
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

const targets = scanRoots.map(buildTarget)

/**
 * Wrap DOM elements in a container.
 * @param {HTMLElement[]} targetDoms
 * @return {HTMLElement}
 */
function domTargetWrapper(targetDoms) {
    const res = document.createElement('div')
    res.className = 'targets-container'
    res.append(...targetDoms)
    return res
}

const app = document.getElementById('app')
if (app) {
    const doms = targets.map(({root}) => {
        const container = document.createElement('ul')
        container.appendChild(root.dom)
        return container
    })
    app.replaceChildren(domTargetWrapper(doms))
}
