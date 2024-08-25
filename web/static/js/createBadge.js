document.getElementById("badge-form").onsubmit = async function (event) {
  event.preventDefault();

  const uniqueName = document.getElementById("unique-name").value;
  const badgeName = document.getElementById("badge-name").value;
  const badgeDescription = document.getElementById("badge-description").value;
  const badgeImage = document.getElementById("badge-image").value;
  const badgeThumb = document.getElementById("badge-thumb").value;

  const badgeEvent = {
    kind: 30009, // Badge Definition kind
    tags: [
      ["d", uniqueName],
      ["name", badgeName]
      ["description", badgeDescription],
      ["image", badgeImage, "1024x1024"],
      ["thumb", badgeThumb, "256x256"],
    ],
    created_at: Math.floor(Date.now() / 1000),
    content: "",
  };

  if (window.nostr) {
    try {
      const signedEvent = await window.nostr.signEvent(badgeEvent);
      console.log("Signed Event:", signedEvent);

      // Send signed event to Go backend
      fetch("/create-badge", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(signedEvent),
      })
        .then((response) => response.json())
        .then((data) => console.log("Badge sent:", data))
        .catch((error) => console.error("Error:", error));
    } catch (err) {
      console.error("Failed to sign event:", err);
    }
  } else {
    alert("Nostr extension not available.");
  }
};
