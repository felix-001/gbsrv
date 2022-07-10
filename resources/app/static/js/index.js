let index = {
    about: function(html) {
        let c = document.createElement("div");
        c.innerHTML = html;
        asticode.modaler.setContent(c);
        asticode.modaler.show();
    },
    init: function() {
        //asticode.loader.init();
        asticode.modaler.init();
        //asticode.notifier.init();

        document.addEventListener('astilectron-ready', function() {
            index.listen();
        })
    },
    listen: function() {
        astilectron.onMessage(function(message) {
            switch (message.name) {
                case "about":
                    index.about(message.payload);
                    return {payload: "payload"};
                case "msg":
                    let div = document.getElementById("main");
                    div.innerHTML = message.payload
                    break
                
            }
        });
    }
};