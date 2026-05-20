"""SM-2 spaced repetition algorithm."""

_SM2_MIN_EF = 1.3
_SM2_EASY_BONUS = 1.3
_SM2_HARD_MULTIPLIER = 1.2
_SM2_HARD_EF_PENALTY = 0.15
_SM2_EASY_EF_BONUS = 0.15


def _sm2_schedule(button, repetitions, interval_days, ease_factor):
    """Calculate new SM-2 scheduling values after a review.

    button: 0=Again (forgot), 1=Hard, 2=Good, 3=Easy
    Returns: (new_repetitions, new_interval_days, new_ease_factor)
    """
    q_map = {0: 0, 1: 2, 2: 4, 3: 5}
    q = q_map.get(button, 2)

    new_ef = ease_factor + (0.1 - (5 - q) * (0.08 + (5 - q) * 0.02))
    new_ef = max(_SM2_MIN_EF, round(new_ef, 2))

    if button == 0:  # Again — complete reset
        return (0, 0, new_ef)

    if button == 1:  # Hard — short interval
        new_int = max(1, round(interval_days * _SM2_HARD_MULTIPLIER)) if interval_days > 0 else 1
        new_ef -= _SM2_HARD_EF_PENALTY
        new_ef = max(_SM2_MIN_EF, round(new_ef, 2))
        return (repetitions, new_int, new_ef)

    if button == 2:  # Good — standard SM-2
        if repetitions == 0:
            new_int = 1
        elif repetitions == 1:
            new_int = 6
        else:
            new_int = round(interval_days * ease_factor)
        new_int = max(1, new_int)
        return (repetitions + 1, new_int, new_ef)

    # button == 3  Easy — with bonus multiplier
    if repetitions == 0:
        new_int = 1
    elif repetitions == 1:
        new_int = 6
    else:
        new_int = round(interval_days * ease_factor)
    new_int = round(new_int * _SM2_EASY_BONUS)
    new_int = max(1, new_int)
    new_ef += _SM2_EASY_EF_BONUS
    new_ef = round(new_ef, 2)
    return (repetitions + 1, new_int, new_ef)
