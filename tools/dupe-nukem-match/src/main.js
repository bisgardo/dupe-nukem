import './style.css'

// Load scan result files.
import testResult1 from '../gendata/test1.json'
import testResult2 from '../gendata/test2.json'

/**
 * The result of scanning the root directory.
 * @typedef {Object} ScanResult
 * @property {number} schema_version
 * @property {ScanDir} root
 */
/**
 * The result of scanning a directory.
 * @typedef {Object} ScanDir
 * @property {string} name
 * @property {ScanDir[]=} dirs
 * @property {ScanFile[]=} files
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

/** @type {ScanDir[]} */
const roots = [testResult1.root, testResult2.root]

/**
 * Index of a target.
 * @typedef {Map<number, TargetFile[]>} Index
 */

/**
 * All processed information of a root.
 * @typedef {Object} Target
 * @property {TargetDir} root
 * @property {Index} index
 */
/**
 * A directory in a target.
 * @typedef {Object} TargetDir
 * @property {TargetDir=} parent
 * @property {ScanDir} scanDir
 * @property {HTMLLIElement} ownDom DOM element of the directory.
 * @property {HTMLUListElement} containerDom DOM element of the container holding the children.
 */
/**
 * A file in a target.
 * @typedef {Object} TargetFile
 * @property {TargetDir} dir
 * @property {ScanFile} scanFile
 * @property {HTMLLIElement} ownDom DOM element of the file.
 */

/**
 * Create new {@link TargetDir}, including the DOM element into which it will be rendered.
 * @param {TargetDir|undefined} parent
 * @param {ScanDir} scanDir
 * @returns TargetDir
 */
function makeTargetDir(parent, scanDir) {
    const ownDom = document.createElement('li')
    ownDom.textContent = scanDir.name
    parent?.containerDom.appendChild(ownDom) // attach to parent's container DOM element
    const containerDom = document.createElement('ul')
    ownDom.appendChild(containerDom) // attach to "own" DOM element
    return {parent, scanDir, ownDom, containerDom}
}

/**
 * Create new {@link TargetFile}, including the DOM element into which it will be rendered.
 * @param {TargetDir} dir
 * @param {ScanFile} scanFile
 * @returns TargetFile
 */
function makeTargetFile(dir, scanFile) {
    const ownDom = document.createElement('li');
    ownDom.textContent = scanFile.name
    dir.containerDom.appendChild(ownDom) // attach to parent's container DOM element
    return {dir, scanFile, ownDom}
}

/**
 * Build target from a scan result.
 * @param {ScanDir} scanRoot
 * @returns {Target}
 */
function buildTarget(scanRoot) {
    /** @type {Index} */
    const index = new Map()

    /**
     * @param {ScanDir} scanDir
     * @param {TargetDir|undefined} parent
     */
    function buildRecursive(scanDir, parent) {
        const targetDir = makeTargetDir(parent, scanDir)
        if (scanDir.dirs !== undefined) {
            for (const d of scanDir.dirs) {
                buildRecursive(d, targetDir)
            }
        }
        if (scanDir.files !== undefined) {
            for (const f of scanDir.files) {
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

const targets = roots.map(buildTarget)

const app = document.getElementById('app');
if (app) {
    app.innerHTML = `<pre>${JSON.stringify(targets, null, 2)}</pre>`
    app.append(...targets.map(({root}) => root.ownDom))
}
