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
        const container = document.createElement('ul')
        container.appendChild(root.dom.self)
        return container
    })
    app.replaceChildren(domTargetWrapper(doms))
}
