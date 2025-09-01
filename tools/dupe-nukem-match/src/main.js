import './style.css'
import {buildTarget} from './target'

// Load scan result files.
import testResult1 from '../gendata/test1.json'
import testResult2 from '../gendata/test2.json'

/** @type {unknown[]} */
const scanRoots = [testResult1.root, testResult2.root]

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
        const innerContainer = document.createElement('ul')
        innerContainer.appendChild(root.dom.self)
        const outerContainer = document.createElement('div')
        outerContainer.appendChild(innerContainer)
        outerContainer.className = 'target-container'
        return outerContainer
    })
    app.replaceChildren(domTargetWrapper(doms))
}
