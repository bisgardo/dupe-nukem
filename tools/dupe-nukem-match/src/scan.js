/**
 * Type definitions and type checking functions for the scan result input types.
 */

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
 * @property {number} hash
 */

/**
 * Determine whether provided value is a non-null object (record).
 * @param {unknown} v
 * @returns {v is Record<string, unknown>}
 */
function isRecord(v) {
    return typeof v === 'object' && v !== null
}

/**
 * Check whether provided value is an array of strings.
 * @param {unknown} v
 * @returns {v is string[]}
 */
function isStringArray(v) {
    return Array.isArray(v) && v.every(x => typeof x === 'string')
}

/**
 * Assert that provided value is a valid {@link ScanFile}, otherwise throw.
 * @param {unknown} scanFile
 * @returns {asserts scanFile is ScanFile}
 */
export function assertScanFile(scanFile) {
    if (!isRecord(scanFile)) throw new TypeError('invalid ScanFile: not an object')
    const {name, size, hash} = scanFile
    if (typeof name !== 'string') throw new TypeError('invalid ScanFile.name')
    if (!Number.isFinite(size)) throw new TypeError('invalid ScanFile.size')
    if (!Number.isFinite(hash)) throw new TypeError('invalid ScanFile.hash')
}

/**
 * Assert (non-recursively) that provided value is a valid {@link ScanDir}, otherwise throw.
 * @param {unknown} scanDir
 * @returns {asserts scanDir is ScanDir}
 */
export function assertScanDir(scanDir) {
    if (!isRecord(scanDir)) throw new TypeError('invalid ScanDir: not an object')
    const {name, dirs, files, empty_files, skipped_files, skipped_dirs} = scanDir
    if (typeof name !== 'string') throw new TypeError('invalid ScanDir.name')
    if (dirs !== undefined && !Array.isArray(dirs)) throw new TypeError('invalid ScanDir.dirs')
    if (files !== undefined && !Array.isArray(files)) throw new TypeError('invalid ScanDir.files')
    if (empty_files !== undefined && !isStringArray(empty_files)) throw new TypeError('invalid ScanDir.empty_files')
    if (skipped_files !== undefined && !isStringArray(skipped_files)) throw new TypeError('invalid ScanDir.skipped_files')
    if (skipped_dirs !== undefined && !isStringArray(skipped_dirs)) throw new TypeError('invalid ScanDir.skipped_dirs')
}
