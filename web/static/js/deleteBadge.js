async function deleteBadge(badgeID) {
  try {
    // Step 1: Fetch the unsigned deletion event from the backend
    const response = await fetch(`/delete-badge?badge_id=${badgeID}`);
    if (!response.ok) {
      throw new Error(`Failed to fetch unsigned event: ${response.statusText}`);
    }

    const unsignedEvent = await response.json();
    console.log("Unsigned Deletion Event:", unsignedEvent); // Log the unsigned event

    // Step 2: Ensure the Nostr extension is available
    if (!window.nostr) {
      alert("Nostr extension not available.");
      return;
    }

    // Step 3: Sign the event using the Nostr extension
    try {
      const signedEvent = await window.nostr.signEvent(unsignedEvent);
      console.log("Signed Deletion Event:", signedEvent);

      // Step 4: Send the signed event to the backend for broadcasting
      const result = await fetch("/delete-signed-badge", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(signedEvent),
      });

      if (!result.ok) {
        const errorMessage = await result.text();
        throw new Error(`Failed to broadcast event: ${errorMessage}`);
      }

      const data = await result.json();
      console.log("Deletion event broadcasted:", data);
      alert("Badge deleted successfully.");
    } catch (err) {
      console.error("Failed to sign deletion event:", err);
      alert(`Failed to sign the deletion event: ${err.message}`);
    }
  } catch (error) {
    console.error("Error fetching unsigned deletion event:", error);
    alert(`Failed to fetch unsigned deletion event: ${error.message}`);
  }
}
