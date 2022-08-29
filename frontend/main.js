// set the huelio version
(function(version){
    document.querySelector('#version').innerText = 'v' + version
    document.title = 'huelio v' + version
})(require("./package.json").version)

var baseURL = '/api/';
try {
    baseURL = process.env.SERVER_URL || baseURL;
} catch(e) {}

// magically check if we're in hueliod or hueliog
var isHuelioG = false;
try {
    isHuelioG = typeof InjectedHostContext === 'function'; 
} catch(e) {}

if (isHuelioG) {
    window.addEventListener('DOMContentLoaded', () => {
        document.querySelectorAll('.hueliog').forEach(e => e.classList.remove('hueliog'));
    });
}

(function() {
    var storage = window.localStorage;
    if(!storage) {
        return;
    }

    if(localStorage.getItem('hideWelcome') !== 'true') {
        document.querySelector('.welcome').setAttribute('style', 'display: initial;');
    }
})();

// Changing the search placeholder regularly
window.setInterval(() => {
    var placeholders = ['link','<room> <scene>', '<room> <color>', '<scene>', '<room> on', '<room> off', 'off', 'on']

    var pick = Math.trunc(Math.random() * placeholders.length - 1)

    document.querySelector('.search input').setAttribute('placeholder', placeholders[pick])

}, 3000)

document.onkeydown = function checkKey(e) {
    switch(e.key) {
        case 'ArrowUp':
        case 'ArrowDown':
            changeSelection(e.key);
            break;

        case 'Enter':
            runSelected().then(response => {
                if(!response.ok) {
                    throw new Error('POST response not OK')
                } else {
                    resetSearch();
                }
            }).catch(error => {
                console.error('Fetch failed', error)
            });
            break;


        default:
            search();
            break;
    }
}

document.onkeyup = function checkKeyUp(e) {
    switch(e.key) {
        case 'Backspace':
            var term = document.querySelector('.search input').value

            if(!term) {
                document.querySelector('.results').innerHTML = ''
            }
            break;
    }
}

var handle = -1

function search() {
    if(handle !== -1) {
        window.clearTimeout(handle);
    }

    var term = document.querySelector('.search input').value

    if(!term) {
        document.querySelector('.results').innerHTML = ''
    }

    handle = window.setTimeout(updateResults, 150)
}


function updateResults(term) {
    var term = document.querySelector('.search input').value

    if(!term) {
        document.querySelector('.results').innerHTML = ''
        return
    }
    
    /**
    handleResults([
        {
            light: {
                name: 'Ceiling'
            },
            onoff: 'off'
        },
        {
            light: {
                name: 'Ceiling'
            },
            onoff: 'on'
        },
        {
            group: {
                name: 'Bed n\' Breakfast'
            },
            onoff: 'on'
        },
        {
            group: {
                name: 'Bed n\' Breakfast'
            },
            scene: {
                name: 'Energize'
            }
        },
    ])
    */

    fetch(baseURL + '?query=' + encodeURIComponent(term)).then(res => res.json()).then(data => handleResults(data))
}

function handleResults(resultsArray) {
    // TODO: If a lamp has the same name in multiple rooms, prefix their names

    var oldResults = document.querySelector('.results')
    var resultDivs = []

    for (let i = 0; i < resultsArray.length; i++) {
        const e = resultsArray[i];
        resultDivs.push(buildResult(e))
    }

    if(resultDivs.length) {
        resultDivs[0].classList.add('active')
    }

    var newResults = document.createElement('div')
    newResults.classList.add('results')

    newResults.append(...resultDivs)

    oldResults.replaceWith(newResults)
}

function buildResult(obj) {
    var result = document.createElement('div')
    result.classList.add('result')
    result.setAttribute('data-action', JSON.stringify(obj))


    result.addEventListener('click', function(e) {
        runAction(this.getAttribute('data-action')).then((r) => {
            if(r.ok) {
                resetSearch()
            } else {
                throw new Error('Run action failed: ' + r.body)
            }
        }).catch(e => {
            throw new Error('Fetch failed: ' + e)
        })
    })

    if(obj.special) {
        var special = document.createElement('div')
        special.classList.add('crumb', 'purple')
        special.innerHTML = '<i class="fas fa-cog">&nbsp;</i><span>' + obj.special.data.message + '</span>'
        result.append(special)
        
        return result
    }


    var lightRoom = document.createElement('div')

    if(obj.light) {
        lightRoom.classList.add('crumb', 'yellow')
        lightRoom.innerHTML = '<i class="fas fa-lightbulb"></i>&nbsp;<span>' + obj.light.data.name + '</span>'
    } else {
        lightRoom.classList.add('crumb', 'orange')
        lightRoom.innerHTML = '<i class="fas fa-layer-group"></i>&nbsp;<span>' + obj.group.data.name + '</span>'
    }

    var toggleOrScene = document.createElement('div')
    toggleOrScene.classList.add('crumb')
    if(obj.onoff) {
        if(obj.onoff === 'off') {
            toggleOrScene.innerHTML = '<i class="fas fa-toggle-off">&nbsp;</i><span>Off</span></div>'
            toggleOrScene.classList.add('red')
        } else {
            toggleOrScene.innerHTML = '<i class="fas fa-toggle-on">&nbsp;</i><span>On</span>'
            toggleOrScene.classList.add('green')
        }
    } else if(obj.color) {
        toggleOrScene.innerHTML = '<i class="fas fa-toggle-on">&nbsp;</i><span style="color:'+ obj.color+ '">' + obj.color + '</span>'
        toggleOrScene.classList.add('white')
    } else {
        toggleOrScene.innerHTML = '<i class="fas fa-toggle-on">&nbsp;</i><span>' + obj.scene.data.name + '</span>'
        toggleOrScene.classList.add('blue')
    }

    var arrow = document.createElement('div')
    arrow.innerHTML = '<i class="fas fa-arrow-right"></i>'
    arrow.classList.add('crumb', 'no-border')


    result.append(lightRoom, arrow, toggleOrScene)

    // For debug
    if(obj.debug) {
        var debug = document.createElement('div')
        debug.classList.add('crumb')
        debug.innerHTML = JSON.stringify(obj.debug)
        
        result.append(debug)
    }

    return result
}

function runSelected() {
    var selected = document.querySelector('.result.active');
    if(!selected) {
        return
    }

    var action = selected.getAttribute('data-action')

    return runAction(action)
}

var actionsRun = 0;

function runAction(action) {
    if(++actionsRun === 3) {
        var storage = window.localStorage
        if(storage) {
            storage.setItem('hideWelcome', 'true')
            document.querySelector('.welcome').setAttribute('style', '');
        }
    }

    return fetch(baseURL, {
        method: 'post',
        headers: {
            'Content-Type': 'application/json'
        },
        body: action
    })
}

function changeSelection(key) {
    var results = document.querySelector('.results');
    var resultList = results.querySelectorAll('.result');

    var activeIdx = -1;
    var newIdx = -1;

    for (let i = 0; i < resultList.length; i++) {
        const element = resultList[i];

        if(element.classList.contains('active')) {
            activeIdx = i;
            element.classList.remove('active');
        }
    }

    if(activeIdx !== -1) {
        newIdx = activeIdx;
    }

    switch(key) {
        case 'ArrowUp':
            newIdx--;
            break;

        case 'ArrowDown':
            newIdx++;
            break;
    }

    newIdx = Math.max(0, Math.min(newIdx, resultList.length - 1));

    resultList[newIdx].classList.add('active')
    resultList[newIdx].scrollIntoView(false)
}

function resetSearch() {
    var search = document.querySelector('.search input')
    search.value = ''
    document.querySelector('.results').innerHTML = ''
    search.focus()
}