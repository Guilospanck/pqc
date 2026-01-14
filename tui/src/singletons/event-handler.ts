type Topic =
  | "exit"
  | "update_message_area"
  | "send_message"
  | "update_input_bar"
  | "add_message"
  | "update_users_panel"
  | "update_current_user_text";

type SubscriberId = string;

type Subscriber = {
  id: SubscriberId;
  callback: (value?: unknown) => void;
};

interface IEventHandler {
  subscribe(topic: Topic, subscriber: Subscriber): void;
  unsubscribe(id: SubscriberId, topic: Topic): void;
  notify(topic: Topic, value?: unknown): void;
}

export const EventHandler = (() => {
  let singleton: IEventHandler | null = null;

  return () => {
    if (singleton) return singleton;

    const subscriptions: Map<Topic, Set<Subscriber>> = new Map();

    singleton = {
      subscribe(topic, subscriber) {
        const currentSubs = subscriptions.get(topic);
        if (!currentSubs) {
          subscriptions.set(topic, new Set([subscriber]));
          return;
        }

        currentSubs.add(subscriber);
        subscriptions.set(topic, currentSubs);
      },
      unsubscribe(id, topic) {
        const currentSubs = subscriptions.get(topic);
        if (!currentSubs) return;

        const filtered = Array.from(currentSubs).filter(
          (item) => item.id !== id,
        );

        subscriptions.set(topic, new Set(filtered));
      },
      notify(topic, value) {
        const currentSubs = subscriptions.get(topic);
        if (!currentSubs) return;

        Array.from(currentSubs).forEach((sub) => sub.callback(value));
      },
    };

    return singleton;
  };
})();
