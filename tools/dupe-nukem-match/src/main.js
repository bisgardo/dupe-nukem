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
        throw new Error(`cannot load local scan file: file not found: ${path}`)
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
    // IDEA: Could split up the work such that each target is displayed right after it's loaded
    //       and then call 'updateMatchInfo' and annotate matches only after they've all loaded?
    const scanResults = await loadLocalScanResults(scanResultPaths)
    const scanRoots = scanResults.map(({root}) => root)
    const targets = scanRoots.map(buildTarget)
    for (const t of targets) {
        t.updateMatchInfo(targets)
    }

    const app = document.getElementById('app')
    if (app) {
        const controller = new Controller(targets)
        const doms = targets.map((target) => {
            const res = new TargetContainerDom(target, controller)
            target.root.dom?.appendTo(res) // attach root to target
            return res.root
        })
        app.replaceChildren(domTargetWrapper(doms))
    }
}

start().catch(console.error)
