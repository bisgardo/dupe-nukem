import './style.css'
import {buildTarget} from './domain.js'
import {TargetContainerDom} from "./dom.js"
import {Controller} from "./controller.js"

const scanResultPaths = [
    '../gendata/test1.json',
    '../gendata/test2.json',
]

/**
 * Wrap DOM elements in a container.
 * @param {HTMLElement[]} targetDoms
 * @return {HTMLElement}
 */
function domTargetWrapper(targetDoms) {
    const targetsContainer = document.createElement('div')
    targetsContainer.className = 'targets-container'
    targetsContainer.replaceChildren(...targetDoms)
    return targetsContainer
}

/**
 * @param {string} path
 * @return {Promise<unknown>}
 */
async function loadLocalScanFile(path) {
    const res = await fetch(path)
    if (!res.ok) {
        throw new TypeError(`cannot load local scan file: file not found: ${path}`)
    }
    return res.json()
}

/**
 * @param {string[]} paths
 * @returns {Promise<import('./scan.js').ScanResult[]>}
 */
async function loadLocalScanResults(paths) {
    // IDEA: Load concurrently in different web workers?
    // @ts-ignore
    return Promise.all(paths.map(loadLocalScanFile))
}

async function start() {
    // TODO: Instead of loading the scan files, we should have it run through a processor first to produce a match file.
    //       This enables reducing hashes to a small number (using a simple counter) to ensure
    //       (1) that we don't get into trouble with JS not implementing 64 bit integers
    //       (2) that we have a place to actually check that matching files are indeed identical.
    // IDEA: Could split up the work such that each target is displayed right after it's loaded
    //       and then call 'updateMatchInfo' and annotate matches only after they've all loaded?
    const time0 = new Date()
    const scanResults = await loadLocalScanResults(scanResultPaths)
    const scanRoots = scanResults.map(({root}) => root)
    const time1 = new Date()
    console.info(`scan result files loaded in ${time1.getTime() - time0.getTime()}ms`)
    const targets = scanRoots.map(buildTarget)
    const time2 = new Date()
    console.info(`targets built in ${time2.getTime() - time1.getTime()}ms`)
    for (const t of targets) {
        t.refreshMatchState(targets.filter(target => target !== t))
    }
    const time3 = new Date()
    console.info(`match state populated in ${time3.getTime() - time2.getTime()}ms`)
    for (const t of targets) {
        t.syncDom()
    }
    const time4 = new Date()
    console.info(`DOM synced in ${time4.getTime() - time3.getTime()}ms`)

    const app = document.getElementById('app')
    if (app) {
        const controller = new Controller(targets)
        const doms = targets.map((target) => {
            console.log(target)
            const res = new TargetContainerDom(target, controller)
            target.root.dom?.appendTo(res) // attach root to target
            return res.root
        })
        app.replaceChildren(domTargetWrapper(doms))
    }
    const time5 = new Date()
    console.info(`all done in a total of ${time5.getTime() - time0.getTime()}ms`)
}

start().catch(console.error)
