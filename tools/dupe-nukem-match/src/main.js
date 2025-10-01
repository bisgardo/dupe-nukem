import './style.css'
import {buildTarget} from './domain.js'
import {TargetContainerDom} from "./dom.js"
import {Controller} from "./controller.js"

let scanResultPaths = [
    '../gendata/test1.json',
    '../gendata/test2.json',
]

/**
 * Wrap DOM elements in a container.
 * @param {HTMLElement[]} targetDoms
 * @return {HTMLElement}
 */
function domTargetWrapper(targetDoms) {
    let targetsContainer = document.createElement('div')
    targetsContainer.className = 'targets-container'
    targetsContainer.replaceChildren(...targetDoms)
    return targetsContainer
}

/**
 * @param {string} path
 * @return {Promise<unknown>}
 */
async function loadLocalScanFile(path) {
    let res = await fetch(path)
    if (!res.ok) {
        throw new TypeError(`cannot load local scan file: file not found: ${path}`)
    }
    return res.json()
}

/**
 * @param {string[]} paths
 * @returns {Promise<import('./scan.js').ScanResult[]>}
 */
function loadLocalScanResults(paths) {
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
    let time0 = new Date()
    let scanResults = await loadLocalScanResults(scanResultPaths)
    let scanRoots = scanResults.map(({root}) => root)
    let time1 = new Date()
    console.info(`scan result files loaded in ${time1.getTime() - time0.getTime()}ms`)
    let targets = scanRoots.map(buildTarget)
    let time2 = new Date()
    console.info(`targets built in ${time2.getTime() - time1.getTime()}ms`)
    for (let t of targets) {
        t.refreshMatchState(targets.filter(target => target !== t))
    }
    let time3 = new Date()
    console.info(`match state populated in ${time3.getTime() - time2.getTime()}ms`)
    for (let t of targets) {
        t.syncDom()
    }
    let time4 = new Date()
    console.info(`DOM synced in ${time4.getTime() - time3.getTime()}ms`)

    let app = document.getElementById('app')
    if (app) {
        let controller = new Controller(targets)
        let doms = targets.map((target) => {
            console.log(target)
            let res = new TargetContainerDom(target, controller)
            target.root.dom?.appendTo(res) // attach root to target
            return res.root
        })
        app.replaceChildren(domTargetWrapper(doms))
    }
    let time5 = new Date()
    console.info(`all done in a total of ${time5.getTime() - time0.getTime()}ms`)
}

start().catch(console.error)
