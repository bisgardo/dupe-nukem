import './style.css'
import {buildTarget} from './domain.js'

// Load scan result files.
import testResult1 from '../gendata/test1.json'
import testResult2 from '../gendata/test2.json'
import {Controller, TargetContainerDom} from "./dom.js";

/** @type {unknown[]} */
const scanRoots = [testResult1.root, testResult2.root]

const targets = scanRoots.map(buildTarget)

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
