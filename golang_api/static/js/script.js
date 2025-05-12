function setVars(data) {

    console.log(data)
    
    const x = document.getElementById("stream").selectedIndex = indStream;
}

window.onload = function() {

    var data = {{ data }};
    setVars(data);
}
